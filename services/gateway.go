package services

import (
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

type GatewayStatus struct {
	Name               string
	URL                string
	LastHealthyTime    time.Time
	LastFailureTime    time.Time
	ConsecutiveFailures int
	State              string // "healthy", "degraded", "unavailable"
	ResponseTimeHistory []time.Duration
	HealthCheckInterval time.Duration
}

type GatewayService struct {
	redisClient     *redis.Client
	httpClient      *http.Client
	gateways        map[string]*GatewayStatus
	mu              sync.RWMutex
	localCache      map[string]gatewayCache
	localCacheMu    sync.RWMutex
}

type gatewayCache struct {
	status    string
	expireAt  time.Time
}

func NewGatewayService(redisClient *redis.Client) *GatewayService {
	now := time.Now()
	return &GatewayService{
		redisClient: redisClient,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
		gateways: map[string]*GatewayStatus{
			"default": {
				Name:                "default",
				URL:                 "http://payment-processor-default:8080",
				LastHealthyTime:     now,
				LastFailureTime:     time.Time{},
				ConsecutiveFailures: 0,
				State:               "healthy",
				ResponseTimeHistory: make([]time.Duration, 0, 10),
				HealthCheckInterval: 6 * time.Second,
			},
			"fallback": {
				Name:                "fallback",
				URL:                 "http://payment-processor-fallback:8080",
				LastHealthyTime:     now,
				LastFailureTime:     time.Time{},
				ConsecutiveFailures: 0,
				State:               "healthy",
				ResponseTimeHistory: make([]time.Duration, 0, 10),
				HealthCheckInterval: 6 * time.Second,
			},
		},
		localCache: make(map[string]gatewayCache),
	}
}

func (g *GatewayService) StartHealthChecker(ctx context.Context) {
	for gatewayName := range g.gateways {
		go g.smartHealthCheckLoop(ctx, gatewayName)
	}
}

func (g *GatewayService) smartHealthCheckLoop(ctx context.Context, gatewayName string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			g.mu.RLock()
			gateway := g.gateways[gatewayName]
			interval := gateway.HealthCheckInterval
			g.mu.RUnlock()

			g.smartHealthCheck(ctx, gatewayName)
			
			time.Sleep(interval)
		}
	}
}

func (g *GatewayService) smartHealthCheck(ctx context.Context, gatewayName string) {
	start := time.Now()
	
	g.mu.RLock()
	gateway := g.gateways[gatewayName]
	g.mu.RUnlock()
	
	healthURL := fmt.Sprintf("%s/payments/service-health", gateway.URL)
	log.Printf("SMART_HEALTH_CHECK: %s state=%s failures=%d", gatewayName, gateway.State, gateway.ConsecutiveFailures)
	
	timeout := g.calculateAdaptiveTimeout(gateway)
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(requestCtx, "GET", healthURL, nil)
	if err != nil {
		g.handleHealthCheckFailure(ctx, gatewayName, fmt.Sprintf("Failed to create request: %v", err))
		return
	}
	
	req.Header.Set("X-Rinha-Token", "123")

	resp, err := g.httpClient.Do(req)
	responseTime := time.Since(start)
	
	if err != nil {
		g.handleHealthCheckFailure(ctx, gatewayName, fmt.Sprintf("Request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		log.Printf("HEALTH_CHECK_RATE_LIMITED: %s - adjusting interval", gatewayName)
		g.adjustHealthCheckInterval(gatewayName, true)
		return
	}

	if resp.StatusCode != http.StatusOK {
		g.handleHealthCheckFailure(ctx, gatewayName, fmt.Sprintf("HTTP %d", resp.StatusCode))
		return
	}

	var healthResponse models.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
		g.handleHealthCheckFailure(ctx, gatewayName, fmt.Sprintf("Failed to decode response: %v", err))
		return
	}

	if healthResponse.Failing {
		g.handleHealthCheckFailure(ctx, gatewayName, "Gateway reports failing status")
		return
	}

	g.handleHealthCheckSuccess(ctx, gatewayName, responseTime)
}

func (g *GatewayService) calculateAdaptiveTimeout(gateway *GatewayStatus) time.Duration {
	baseTimeout := 5 * time.Second
	
	if len(gateway.ResponseTimeHistory) > 0 {
		total := time.Duration(0)
		for _, rt := range gateway.ResponseTimeHistory {
			total += rt
		}
		avgResponseTime := total / time.Duration(len(gateway.ResponseTimeHistory))
		
		adaptiveTimeout := avgResponseTime*3 + 2*time.Second
		
		if adaptiveTimeout < 3*time.Second {
			adaptiveTimeout = 3 * time.Second
		}
		if adaptiveTimeout > 10*time.Second {
			adaptiveTimeout = 10 * time.Second
		}
		
		return adaptiveTimeout
	}
	
	return baseTimeout
}

func (g *GatewayService) handleHealthCheckSuccess(ctx context.Context, gatewayName string, responseTime time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	gateway := g.gateways[gatewayName]
	now := time.Now()
	
	gateway.LastHealthyTime = now
	gateway.ConsecutiveFailures = 0
	oldState := gateway.State
	gateway.State = "healthy"
	
	gateway.ResponseTimeHistory = append(gateway.ResponseTimeHistory, responseTime)
	if len(gateway.ResponseTimeHistory) > 10 {
		gateway.ResponseTimeHistory = gateway.ResponseTimeHistory[1:]
	}
	
	g.adjustHealthCheckIntervalUnlocked(gatewayName, false)
	
	key := fmt.Sprintf("gateway:%s", gatewayName)
	ttl := g.calculateRedisTTL(gateway)
	
	err := g.redisClient.Set(ctx, key, "healthy", ttl).Err()
	if err != nil {
		log.Printf("Failed to set gateway %s as healthy in Redis: %v", gatewayName, err)
	} else {
		if oldState != "healthy" {
			log.Printf("GATEWAY_RECOVERED: %s %s→healthy responseTime=%v", gatewayName, oldState, responseTime)
		} else {
			log.Printf("GATEWAY_HEALTHY: %s responseTime=%v ttl=%v", gatewayName, responseTime, ttl)
		}
	}
}

func (g *GatewayService) handleHealthCheckFailure(ctx context.Context, gatewayName string, reason string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	gateway := g.gateways[gatewayName]
	now := time.Now()
	
	gateway.LastFailureTime = now
	gateway.ConsecutiveFailures++
	oldState := gateway.State
	
	if gateway.ConsecutiveFailures >= 3 {
		gateway.State = "unavailable"
		g.removeGatewayFromCacheUnlocked(ctx, gatewayName)
	} else if gateway.ConsecutiveFailures >= 1 && time.Since(gateway.LastHealthyTime) > 30*time.Second {
		gateway.State = "degraded"
		g.setDegradedGatewayInRedis(ctx, gatewayName)
	} else {
		log.Printf("GATEWAY_GRACE_PERIOD: %s failures=%d reason=%s", gatewayName, gateway.ConsecutiveFailures, reason)
		return
	}
	
	g.adjustHealthCheckIntervalUnlocked(gatewayName, true)
	
	if oldState != gateway.State {
		log.Printf("GATEWAY_STATE_CHANGE: %s %s→%s failures=%d reason=%s", 
			gatewayName, oldState, gateway.State, gateway.ConsecutiveFailures, reason)
	}
}

func (g *GatewayService) adjustHealthCheckInterval(gatewayName string, rateLimited bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.adjustHealthCheckIntervalUnlocked(gatewayName, rateLimited)
}

func (g *GatewayService) adjustHealthCheckIntervalUnlocked(gatewayName string, increaseDueToFailure bool) {
	gateway := g.gateways[gatewayName]
	
	if increaseDueToFailure {
		switch gateway.State {
		case "healthy":
			gateway.HealthCheckInterval = 8 * time.Second
		case "degraded":
			gateway.HealthCheckInterval = 15 * time.Second
		case "unavailable":
			gateway.HealthCheckInterval = 30 * time.Second
		}
	} else {
		if gateway.ConsecutiveFailures == 0 && len(gateway.ResponseTimeHistory) >= 3 {
			gateway.HealthCheckInterval = 10 * time.Second
		} else {
			gateway.HealthCheckInterval = 6 * time.Second
		}
	}
	
	log.Printf("HEALTH_CHECK_INTERVAL_ADJUSTED: %s interval=%v state=%s", 
		gatewayName, gateway.HealthCheckInterval, gateway.State)
}

func (g *GatewayService) calculateRedisTTL(gateway *GatewayStatus) time.Duration {
	baseTTL := 15 * time.Second
	
	if gateway.ConsecutiveFailures == 0 && len(gateway.ResponseTimeHistory) >= 5 {
		return 25 * time.Second
	}
	
	return baseTTL
}

func (g *GatewayService) setDegradedGatewayInRedis(ctx context.Context, gatewayName string) {
	key := fmt.Sprintf("gateway:%s", gatewayName)
	err := g.redisClient.Set(ctx, key, "degraded", 8*time.Second).Err()
	if err != nil {
		log.Printf("Failed to set degraded gateway %s in Redis: %v", gatewayName, err)
	}
}

func (g *GatewayService) removeGatewayFromCacheUnlocked(ctx context.Context, gatewayName string) {
	key := fmt.Sprintf("gateway:%s", gatewayName)
	err := g.redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to remove gateway %s from Redis: %v", gatewayName, err)
	}
}

func (g *GatewayService) GetAvailableGateway(ctx context.Context) (string, error) {
	return g.GetAvailableGatewayWithProfiling(ctx, "")
}

func (g *GatewayService) GetAvailableGatewayWithProfiling(ctx context.Context, correlationID string) (string, error) {
	profile := &models.GatewaySelectionProfile{
		CorrelationID: correlationID,
		StartTime:     time.Now(),
	}
	defer g.logSelectionProfile(profile)
	
	gateways := []string{"default", "fallback"}
	
	for _, gatewayName := range gateways {
		if cached := g.getFromLocalCache(gatewayName); cached != "" {
			profile.TotalTime = time.Since(profile.StartTime)
			profile.SelectedGateway = gatewayName
			profile.SelectionMethod = "local_cache"
			
			log.Printf("GATEWAY_SELECTED_LOCAL: gateway=%s status=%s time=%v", gatewayName, cached, profile.TotalTime)
			return gatewayName, nil
		}
	}
	
	redisStart := time.Now()
	gatewayStatuses := g.getBatchGatewayStatus(ctx, gateways)
	profile.RedisHits = len(gateways)
	
	for _, gatewayName := range gateways {
		if status, exists := gatewayStatuses[gatewayName]; exists {
			g.setLocalCache(gatewayName, status, 3*time.Second)
			
			profile.RedisLookupTime = time.Since(redisStart)
			profile.TotalTime = time.Since(profile.StartTime)
			profile.SelectedGateway = gatewayName
			profile.SelectionMethod = "redis_cache"
			
			log.Printf("GATEWAY_SELECTED: gateway=%s status=%s time=%v", gatewayName, status, profile.TotalTime)
			return gatewayName, nil
		}
	}
	profile.RedisLookupTime = time.Since(redisStart)
	
	log.Printf("GATEWAY_FALLBACK_TO_LAST_KNOWN: checking historical availability")
	historyStart := time.Now()
	profile.HistoryChecked = true
	bestGateway := g.getBestAvailableFromHistory()
	profile.HistoryLookupTime = time.Since(historyStart)
	
	if bestGateway != "" {
		profile.TotalTime = time.Since(profile.StartTime)
		profile.SelectedGateway = bestGateway
		profile.SelectionMethod = "history_cache"
		
		log.Printf("GATEWAY_SELECTED_FROM_HISTORY: gateway=%s time=%v", bestGateway, profile.TotalTime)
		return bestGateway, nil
	}
	
	gracePeriodStart := time.Now()
	profile.GracePeriodUsed = true
	gracePeriodGateway := g.getGracePeriodGateway()
	profile.GracePeriodTime = time.Since(gracePeriodStart)
	
	if gracePeriodGateway != "" {
		profile.TotalTime = time.Since(profile.StartTime)
		profile.SelectedGateway = gracePeriodGateway
		profile.SelectionMethod = "grace_period"
		
		log.Printf("GATEWAY_SELECTED_GRACE_PERIOD: gateway=%s time=%v", gracePeriodGateway, profile.TotalTime)
		return gracePeriodGateway, nil
	}
	
	profile.TotalTime = time.Since(profile.StartTime)
	profile.SelectionMethod = "failed"
	
	log.Printf("GATEWAY_UNAVAILABLE: all_gateways_exhausted time=%v", profile.TotalTime)
	return "", fmt.Errorf("no gateway available")
}

func (g *GatewayService) getFromLocalCache(gatewayName string) string {
	g.localCacheMu.RLock()
	defer g.localCacheMu.RUnlock()
	
	if cached, exists := g.localCache[gatewayName]; exists {
		if time.Now().Before(cached.expireAt) {
			return cached.status
		}
		delete(g.localCache, gatewayName)
	}
	return ""
}

func (g *GatewayService) setLocalCache(gatewayName, status string, ttl time.Duration) {
	g.localCacheMu.Lock()
	defer g.localCacheMu.Unlock()
	
	g.localCache[gatewayName] = gatewayCache{
		status:   status,
		expireAt: time.Now().Add(ttl),
	}
}

func (g *GatewayService) getBatchGatewayStatus(ctx context.Context, gateways []string) map[string]string {
	pipe := g.redisClient.Pipeline()
	
	commands := make(map[string]*redis.StringCmd)
	for _, gatewayName := range gateways {
		key := fmt.Sprintf("gateway:%s", gatewayName)
		commands[gatewayName] = pipe.Get(ctx, key)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		log.Printf("Redis pipeline error: %v", err)
		return make(map[string]string)
	}
	
	result := make(map[string]string)
	for gatewayName, cmd := range commands {
		if val, err := cmd.Result(); err == nil {
			result[gatewayName] = val
		}
	}
	
	return result
}

func (g *GatewayService) logSelectionProfile(profile *models.GatewaySelectionProfile) {
	if profile.TotalTime > 50*time.Millisecond {
		log.Printf("GATEWAY_SELECTION_SLOW: correlationId=%s total=%v redis=%v history=%v grace=%v method=%s gateway=%s hits=%d",
			profile.CorrelationID,
			profile.TotalTime,
			profile.RedisLookupTime,
			profile.HistoryLookupTime,
			profile.GracePeriodTime,
			profile.SelectionMethod,
			profile.SelectedGateway,
			profile.RedisHits)
	}
}

func (g *GatewayService) getBestAvailableFromHistory() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	var bestGateway string
	var bestTime time.Time
	
	cutoff := time.Now().Add(-5 * time.Minute)
	
	for name, gateway := range g.gateways {
		if gateway.LastHealthyTime.After(cutoff) && gateway.LastHealthyTime.After(bestTime) {
			bestTime = gateway.LastHealthyTime
			bestGateway = name
		}
	}
	
	return bestGateway
}

func (g *GatewayService) getGracePeriodGateway() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	for name, gateway := range g.gateways {
		if gateway.State == "degraded" || 
		   (gateway.ConsecutiveFailures <= 2 && time.Since(gateway.LastHealthyTime) < 2*time.Minute) {
			return name
		}
	}
	
	return "default"
}

func (g *GatewayService) GetGatewayURL(gatewayName string) string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	if gateway, exists := g.gateways[gatewayName]; exists {
		return gateway.URL
	}
	return ""
}