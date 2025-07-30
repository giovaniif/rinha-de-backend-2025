package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// ProcessorInfo representa informações de um processor em cache
type ProcessorInfo struct {
	URL         string    `json:"url"`
	Name        string    `json:"name"`
	IsDefault   bool      `json:"is_default"`
	IsAvailable bool      `json:"is_available"`
	LastCheck   time.Time `json:"last_check"`
}

// RedisCache gerencia o cache de processors no Redis
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

const (
	// Chaves do Redis
	CACHE_KEY_AVAILABLE_GATEWAY = "rinha:available_gateway"
	CACHE_KEY_DEFAULT_STATUS    = "rinha:default_status"
	CACHE_KEY_FALLBACK_STATUS   = "rinha:fallback_status"
	
	// TTL do cache
	CACHE_TTL = 30 * time.Second
)

// NewRedisCache cria uma nova instância do cache Redis
func NewRedisCache(redisURL string) (*RedisCache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear Redis URL: %v", err)
	}

	client := redis.NewClient(opts)
	ctx := context.Background()

	// Testar conexão
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("erro ao conectar com Redis: %v", err)
	}

	log.Printf("✅ Redis Cache conectado com sucesso: %s", redisURL)

	return &RedisCache{
		client: client,
		ctx:    ctx,
	}, nil
}

// GetAvailableGateway retorna o último gateway disponível do cache
func (r *RedisCache) GetAvailableGateway() (*ProcessorInfo, error) {
	data, err := r.client.Get(r.ctx, CACHE_KEY_AVAILABLE_GATEWAY).Result()
	if err == redis.Nil {
		log.Printf("🔍 Cache miss: nenhum gateway disponível no cache")
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar gateway no cache: %v", err)
	}

	var processor ProcessorInfo
	if err := json.Unmarshal([]byte(data), &processor); err != nil {
		return nil, fmt.Errorf("erro ao deserializar gateway do cache: %v", err)
	}

	log.Printf("📋 Cache hit: gateway %s (%s) - última verificação: %v", 
		processor.Name, processor.URL, processor.LastCheck.Format(time.RFC3339))
	
	return &processor, nil
}

// SetAvailableGateway armazena o gateway disponível no cache
func (r *RedisCache) SetAvailableGateway(processor *ProcessorInfo) error {
	processor.LastCheck = time.Now()
	
	data, err := json.Marshal(processor)
	if err != nil {
		return fmt.Errorf("erro ao serializar gateway para cache: %v", err)
	}

	err = r.client.Set(r.ctx, CACHE_KEY_AVAILABLE_GATEWAY, data, CACHE_TTL).Err()
	if err != nil {
		return fmt.Errorf("erro ao salvar gateway no cache: %v", err)
	}

	log.Printf("💾 Gateway %s (%s) salvo no cache (TTL: %v)", 
		processor.Name, processor.URL, CACHE_TTL)
	
	return nil
}

// InvalidateGateway remove o gateway específico do cache
func (r *RedisCache) InvalidateGateway(processorName string) error {
	// Verificar se o gateway no cache é o que está falhando
	currentGateway, err := r.GetAvailableGateway()
	if err != nil {
		return err
	}
	
	if currentGateway != nil && currentGateway.Name == processorName {
		err = r.client.Del(r.ctx, CACHE_KEY_AVAILABLE_GATEWAY).Err()
		if err != nil {
			return fmt.Errorf("erro ao invalidar gateway %s do cache: %v", processorName, err)
		}
		
		log.Printf("🗑️ Gateway %s invalidado do cache", processorName)
	}
	
	return nil
}

// SetProcessorStatus armazena o status individual de um processor
func (r *RedisCache) SetProcessorStatus(processorName string, isAvailable bool) error {
	key := r.getProcessorStatusKey(processorName)
	
	statusData := map[string]interface{}{
		"name":         processorName,
		"is_available": isAvailable,
		"last_check":   time.Now(),
	}
	
	data, err := json.Marshal(statusData)
	if err != nil {
		return fmt.Errorf("erro ao serializar status do processor: %v", err)
	}
	
	err = r.client.Set(r.ctx, key, data, CACHE_TTL).Err()
	if err != nil {
		return fmt.Errorf("erro ao salvar status do processor no cache: %v", err)
	}
	
	return nil
}

// GetProcessorStatus retorna o status de um processor específico
func (r *RedisCache) GetProcessorStatus(processorName string) (bool, error) {
	key := r.getProcessorStatusKey(processorName)
	
	data, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return false, nil // Cache miss, assume não disponível
	}
	if err != nil {
		return false, fmt.Errorf("erro ao buscar status do processor: %v", err)
	}
	
	var statusData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &statusData); err != nil {
		return false, fmt.Errorf("erro ao deserializar status do processor: %v", err)
	}
	
	isAvailable, ok := statusData["is_available"].(bool)
	if !ok {
		return false, fmt.Errorf("formato inválido do status do processor")
	}
	
	return isAvailable, nil
}

// GetAllProcessorStatus retorna o status de todos os processors
func (r *RedisCache) GetAllProcessorStatus() map[string]bool {
	status := make(map[string]bool)
	
	// Buscar status do default
	if defaultStatus, err := r.GetProcessorStatus("default"); err == nil {
		status["default"] = defaultStatus
	} else {
		status["default"] = false
	}
	
	// Buscar status do fallback
	if fallbackStatus, err := r.GetProcessorStatus("fallback"); err == nil {
		status["fallback"] = fallbackStatus
	} else {
		status["fallback"] = false
	}
	
	return status
}

// Close fecha a conexão com o Redis
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// Helper para gerar chave do status do processor
func (r *RedisCache) getProcessorStatusKey(processorName string) string {
	switch processorName {
	case "default":
		return CACHE_KEY_DEFAULT_STATUS
	case "fallback":
		return CACHE_KEY_FALLBACK_STATUS
	default:
		return fmt.Sprintf("rinha:%s_status", processorName)
	}
} 