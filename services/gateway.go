package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"payment-processor/models"
)

type GatewayService struct {
	redisClient *redis.Client
	httpClient  *http.Client
	gateways    map[string]string
}

func NewGatewayService(redisClient *redis.Client) *GatewayService {
	return &GatewayService{
		redisClient: redisClient,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
		gateways: map[string]string{
			"default":  "http://payment-processor-default:8080",
			"fallback": "http://payment-processor-fallback:8080",
		},
	}
}

func (g *GatewayService) StartHealthChecker(ctx context.Context) {
	for gatewayName, gatewayURL := range g.gateways {
		go g.healthCheckLoop(ctx, gatewayName, gatewayURL)
	}
}

func (g *GatewayService) healthCheckLoop(ctx context.Context, gatewayName, gatewayURL string) {
	ticker := time.NewTicker(6 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.checkHealth(ctx, gatewayName, gatewayURL)
		}
	}
}

func (g *GatewayService) checkHealth(ctx context.Context, gatewayName, gatewayURL string) {
	healthURL := fmt.Sprintf("%s/payments/service-health", gatewayURL)
	log.Printf("Checking health for %s at %s", gatewayName, healthURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		log.Printf("Failed to create request for %s: %v", gatewayName, err)
		g.removeGatewayFromCache(ctx, gatewayName)
		return
	}
	
	req.Header.Set("X-Rinha-Token", "123")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		log.Printf("Health check failed for %s: %v", gatewayName, err)
		g.removeGatewayFromCache(ctx, gatewayName)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Health check returned status %d for %s", resp.StatusCode, gatewayName)
		g.removeGatewayFromCache(ctx, gatewayName)
		return
	}

	var healthResponse models.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
		log.Printf("Failed to decode health response for %s: %v", gatewayName, err)
		g.removeGatewayFromCache(ctx, gatewayName)
		return
	}

	if healthResponse.Failing {
		log.Printf("Gateway %s is failing", gatewayName)
		g.removeGatewayFromCache(ctx, gatewayName)
		return
	}

	key := fmt.Sprintf("gateway:%s", gatewayName)
	err = g.redisClient.Set(ctx, key, "healthy", 12*time.Second).Err()
	if err != nil {
		log.Printf("Failed to set gateway %s as healthy in Redis: %v", gatewayName, err)
	} else {
		log.Printf("Gateway %s marked as healthy", gatewayName)
	}
}

func (g *GatewayService) removeGatewayFromCache(ctx context.Context, gatewayName string) {
	key := fmt.Sprintf("gateway:%s", gatewayName)
	err := g.redisClient.Del(ctx, key).Err()
	if err != nil {
		log.Printf("Failed to remove gateway %s from Redis: %v", gatewayName, err)
	}
}

func (g *GatewayService) GetAvailableGateway(ctx context.Context) (string, error) {
	gateways := []string{"default", "fallback"}
	
	for _, gateway := range gateways {
		key := fmt.Sprintf("gateway:%s", gateway)
		result := g.redisClient.Get(ctx, key)
		if result.Err() == nil {
			log.Printf("GATEWAY_SELECTED: gateway=%s status=available", gateway)
			return gateway, nil
		}
	}
	
	log.Printf("GATEWAY_UNAVAILABLE: all_gateways_down")
	return "", fmt.Errorf("no gateway available")
}

func (g *GatewayService) GetGatewayURL(gatewayName string) string {
	return g.gateways[gatewayName]
}