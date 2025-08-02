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
	redisClient    *redis.Client
	gatewayService *GatewayService
	httpClient     *http.Client
	paymentQueue   chan models.PaymentJob
	mu             sync.RWMutex
	stats          map[string]*models.ProcessorSummary
}

func NewPaymentService(redisClient *redis.Client, gatewayService *GatewayService) *PaymentService {
	return &PaymentService{
		redisClient:    redisClient,
		gatewayService: gatewayService,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		paymentQueue: make(chan models.PaymentJob, 1000),
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
	gatewayType, err := p.gatewayService.GetAvailableGateway(ctx)
	if err != nil {
		return fmt.Errorf("no gateway available: %w", err)
	}

	paymentReq := models.PaymentProcessorRequest{
		CorrelationID: req.CorrelationID,
		Amount:        req.Amount,
		RequestedAt:   time.Now().UTC(),
	}

	job := models.PaymentJob{
		PaymentRequest: paymentReq,
		ProcessorType:  gatewayType,
		Attempts:       0,
		MaxAttempts:    3,
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
	job.Attempts++
	log.Printf("Processing payment (attempt %d): %s", job.Attempts, job.PaymentRequest.CorrelationID)

	gatewayURL := p.gatewayService.GetGatewayURL(job.ProcessorType)
	if gatewayURL == "" {
		log.Printf("Unknown gateway type: %s", job.ProcessorType)
		return
	}

	paymentURL := fmt.Sprintf("%s/payments", gatewayURL)

	jsonData, err := json.Marshal(job.PaymentRequest)
	if err != nil {
		log.Printf("Failed to marshal payment request: %v", err)
		p.retryOrFail(ctx, job)
		return
	}
	
	log.Printf("Sending payment to %s: %s", paymentURL, string(jsonData))

	req, err := http.NewRequestWithContext(ctx, "POST", paymentURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create payment request: %v", err)
		p.retryOrFail(ctx, job)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Rinha-Token", "123")
	
	log.Printf("Request headers: %v", req.Header)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to send payment request: %v", err)
		p.retryOrFail(ctx, job)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		p.updateStats(job.ProcessorType, job.PaymentRequest.Amount)
		log.Printf("Payment processed successfully: %s via %s", job.PaymentRequest.CorrelationID, job.ProcessorType)
	} else {
		log.Printf("Payment failed with status %d: %s", resp.StatusCode, job.PaymentRequest.CorrelationID)
		p.retryOrFail(ctx, job)
	}
}

func (p *PaymentService) retryOrFail(ctx context.Context, job models.PaymentJob) {
	if job.Attempts < job.MaxAttempts {
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