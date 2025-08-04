package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"payment-processor/models"
)

type PaymentService struct {
	redisClient      *redis.Client
	gatewayService   *GatewayService
	httpClient       *http.Client
	paymentQueue     chan models.PaymentJob
	mu               sync.RWMutex
	stats            map[string]*models.ProcessorSummary
	circuitBreakers  *ProcessorCircuitBreakers
}

func NewPaymentService(redisClient *redis.Client, gatewayService *GatewayService) *PaymentService {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
		DisableKeepAlives:   false,
	}

	httpClient := &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}

	log.Printf("HTTP_CLIENT_OPTIMIZED: timeout=15s max_idle_conns=100 idle_timeout=90s")

	return &PaymentService{
		redisClient:     redisClient,
		gatewayService:  gatewayService,
		httpClient:      httpClient,
		paymentQueue:    make(chan models.PaymentJob, 5000),
		circuitBreakers: NewProcessorCircuitBreakers(),
		stats: map[string]*models.ProcessorSummary{
			"default":  {TotalRequests: 0, TotalAmount: 0},
			"fallback": {TotalRequests: 0, TotalAmount: 0},
		},
	}
}

func (p *PaymentService) StartPaymentProcessor(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		go p.paymentWorker(ctx)
	}
}

func (p *PaymentService) paymentWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-p.paymentQueue:
			p.processPayment(ctx, job)
		}
	}
}

func (p *PaymentService) ProcessPaymentRequest(ctx context.Context, req models.PaymentRequest) error {
	paymentReq := models.PaymentProcessorRequest{
		CorrelationID: req.CorrelationID,
		Amount:        req.Amount,
		RequestedAt:   time.Now().UTC(),
	}

	job := models.PaymentJob{
		PaymentRequest: paymentReq,
		ProcessorType:  "",
		Attempts:       0,
		MaxAttempts:    5,
	}

	select {
	case p.paymentQueue <- job:
		log.Printf("Payment enqueued successfully: %s", req.CorrelationID)
		return nil
	default:
		log.Printf("Payment queue is full for: %s", req.CorrelationID)
		return fmt.Errorf("payment queue is full")
	}
}

func (p *PaymentService) processPayment(ctx context.Context, job models.PaymentJob) {
	profile := &models.PaymentProfile{
		CorrelationID:    job.PaymentRequest.CorrelationID,
		StartTime:        time.Now(),
		AttemptNumber:    job.Attempts + 1,
		PaymentProcessor: job.ProcessorType,
	}
	defer p.logPaymentProfile(profile)

	job.Attempts++
	log.Printf("Processing payment (attempt %d): %s", job.Attempts, job.PaymentRequest.CorrelationID)

	if job.ProcessorType == "" {
		gatewaySelectionStart := time.Now()
		gatewayType, err := p.gatewayService.GetAvailableGatewayWithProfiling(ctx, job.PaymentRequest.CorrelationID)
		profile.GatewaySelectionTime = time.Since(gatewaySelectionStart)
		
		if err != nil {
			profile.Success = false
			profile.ErrorType = "no_gateway_available"
			log.Printf("GATEWAY_UNAVAILABLE_RETRY: correlationId=%s attempt=%d", job.PaymentRequest.CorrelationID, job.Attempts)
			p.retryNoGateway(ctx, job)
			return
		}
		job.ProcessorType = gatewayType
		profile.PaymentProcessor = gatewayType
	}

	gatewayURL := p.gatewayService.GetGatewayURL(job.ProcessorType)
	if gatewayURL == "" {
		profile.Success = false
		profile.ErrorType = "unknown_gateway_type"
		log.Printf("Unknown gateway type: %s", job.ProcessorType)
		p.retryOrFail(ctx, job)
		return
	}

	paymentURL := fmt.Sprintf("%s/payments", gatewayURL)

	// JSON Serialization profiling
	jsonStart := time.Now()
	jsonData, err := json.Marshal(job.PaymentRequest)
	profile.JSONSerializationTime = time.Since(jsonStart)
	
	if err != nil {
		profile.Success = false
		profile.ErrorType = "json_marshal_error"
		log.Printf("Failed to marshal payment request: %v", err)
		p.retryOrFail(ctx, job)
		return
	}
	
	log.Printf("Sending payment to %s: %s", paymentURL, string(jsonData))

	req, err := http.NewRequestWithContext(ctx, "POST", paymentURL, bytes.NewBuffer(jsonData))
	if err != nil {
		profile.Success = false
		profile.ErrorType = "http_request_creation_error"
		log.Printf("Failed to create payment request: %v", err)
		p.retryOrFail(ctx, job)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Rinha-Token", "123")
	
	log.Printf("Request headers: %v", req.Header)

	// Circuit breaker protection
	circuitBreaker := p.circuitBreakers.GetOrCreateBreaker(job.ProcessorType)
	
	if !circuitBreaker.CanExecute() {
		profile.Success = false
		profile.ErrorType = "circuit_breaker_open"
		log.Printf("CIRCUIT_BREAKER_BLOCKED: correlationId=%s processor=%s", 
			job.PaymentRequest.CorrelationID, job.ProcessorType)
		p.retryOrFail(ctx, job)
		return
	}

	// HTTP Request profiling with circuit breaker
	httpStart := time.Now()
	var resp *http.Response
	
	err = circuitBreaker.Call(func() error {
		var httpErr error
		resp, httpErr = p.httpClient.Do(req)
		
		if httpErr != nil {
			return httpErr
		}
		
		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: status %d", resp.StatusCode)
		}
		
		return nil
	})
	
	profile.HTTPRequestTime = time.Since(httpStart)
	
	if err != nil {
		profile.Success = false
		if resp != nil {
			profile.StatusCode = resp.StatusCode
			profile.ErrorType = fmt.Sprintf("http_status_%d", resp.StatusCode)
		} else {
			profile.ErrorType = "http_request_error"
		}
		log.Printf("PAYMENT_FAILED_CIRCUIT: correlationId=%s gateway=%s error=%v", 
			job.PaymentRequest.CorrelationID, job.ProcessorType, err)
		p.retryOrFail(ctx, job)
		return
	}
	defer resp.Body.Close()

	profile.StatusCode = resp.StatusCode

	// Response processing
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		profile.Success = true
		p.updateStats(job.ProcessorType, job.PaymentRequest.Amount)
		log.Printf("PAYMENT_SUCCESS: correlationId=%s gateway=%s amount=%.2f attempt=%d status=%d", 
			job.PaymentRequest.CorrelationID, job.ProcessorType, job.PaymentRequest.Amount, job.Attempts, resp.StatusCode)
	} else {
		profile.Success = false
		profile.ErrorType = fmt.Sprintf("http_status_%d", resp.StatusCode)
		log.Printf("PAYMENT_FAILED: correlationId=%s gateway=%s amount=%.2f attempt=%d status=%d", 
			job.PaymentRequest.CorrelationID, job.ProcessorType, job.PaymentRequest.Amount, job.Attempts, resp.StatusCode)
		p.retryOrFail(ctx, job)
	}
	
	profile.TotalTime = time.Since(profile.StartTime)
}

func (p *PaymentService) retryOrFail(ctx context.Context, job models.PaymentJob) {
	if job.Attempts < job.MaxAttempts {
		job.ProcessorType = ""
		time.Sleep(1 * time.Second)
		select {
		case p.paymentQueue <- job:
		default:
			log.Printf("Failed to requeue payment: %s", job.PaymentRequest.CorrelationID)
		}
	} else {
		log.Printf("Payment failed after %d attempts: %s", job.MaxAttempts, job.PaymentRequest.CorrelationID)
	}
}

func (p *PaymentService) retryNoGateway(ctx context.Context, job models.PaymentJob) {
	if job.Attempts >= job.MaxAttempts {
		log.Printf("PAYMENT_FAILED_NO_GATEWAY: correlationId=%s attempts=%d", job.PaymentRequest.CorrelationID, job.Attempts)
		return
	}

	backoffSeconds := []int{2, 5, 10, 15, 30}
	attemptIndex := job.Attempts - 1
	if attemptIndex >= len(backoffSeconds) {
		attemptIndex = len(backoffSeconds) - 1
	}
	
	delay := time.Duration(backoffSeconds[attemptIndex]) * time.Second
	log.Printf("GATEWAY_RETRY_SCHEDULED: correlationId=%s attempt=%d delay=%v", 
		job.PaymentRequest.CorrelationID, job.Attempts, delay)

	go func() {
		time.Sleep(delay)
		select {
		case p.paymentQueue <- job:
			log.Printf("GATEWAY_RETRY_QUEUED: correlationId=%s attempt=%d", 
				job.PaymentRequest.CorrelationID, job.Attempts)
		default:
			log.Printf("GATEWAY_RETRY_QUEUE_FULL: correlationId=%s attempt=%d", 
				job.PaymentRequest.CorrelationID, job.Attempts)
		}
	}()
}

func (p *PaymentService) updateStats(processorType string, amount float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if stats, exists := p.stats[processorType]; exists {
		stats.TotalRequests++
		stats.TotalAmount += amount
	}
}

func (p *PaymentService) GetPaymentsSummary(from, to time.Time) models.PaymentsSummaryResponse {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return models.PaymentsSummaryResponse{
		Default:  *p.stats["default"],
		Fallback: *p.stats["fallback"],
	}
}

func (p *PaymentService) logPaymentProfile(profile *models.PaymentProfile) {
	if profile.TotalTime > 500*time.Millisecond {
		log.Printf("PAYMENT_PROCESSING_SLOW: correlationId=%s total=%v gateway_selection=%v json_serialization=%v http_request=%v processor=%s attempt=%d success=%t status=%d error=%s",
			profile.CorrelationID,
			profile.TotalTime,
			profile.GatewaySelectionTime,
			profile.JSONSerializationTime,
			profile.HTTPRequestTime,
			profile.PaymentProcessor,
			profile.AttemptNumber,
			profile.Success,
			profile.StatusCode,
			profile.ErrorType)
	}
	
	if profile.HTTPRequestTime > 200*time.Millisecond {
		log.Printf("HTTP_REQUEST_SLOW: correlationId=%s http_time=%v processor=%s status=%d",
			profile.CorrelationID,
			profile.HTTPRequestTime,
			profile.PaymentProcessor,
			profile.StatusCode)
	}
	
	if profile.JSONSerializationTime > 10*time.Millisecond {
		log.Printf("JSON_SERIALIZATION_SLOW: correlationId=%s json_time=%v",
			profile.CorrelationID,
			profile.JSONSerializationTime)
	}
}

func (p *PaymentService) GetCircuitBreakerStats() map[string]interface{} {
	p.circuitBreakers.mu.RLock()
	defer p.circuitBreakers.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	for name, breaker := range p.circuitBreakers.breakers {
		state, failureCount, successCount, lastFailTime := breaker.GetStats()
		
		stats[name] = map[string]interface{}{
			"state":        state,
			"failureCount": failureCount,
			"successCount": successCount,
			"lastFailTime": lastFailTime,
		}
	}
	
	return stats
}