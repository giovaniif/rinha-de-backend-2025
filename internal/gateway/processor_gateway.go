package gateway

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"rinha-de-backend-2025/internal/cache"
)

// ProcessorInfo representa informações sobre um payment processor
type ProcessorInfo struct {
	URL       string `json:"url"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// ProcessorGateway gerencia a decisão de qual processor usar (Arquitetura 2 com Redis Cache)
type ProcessorGateway struct {
	defaultURL  string
	fallbackURL string
	httpClient  *http.Client
	redisCache  *cache.RedisCache // Arquitetura 2: Usar Redis cache
}

// NewProcessorGateway cria uma nova instância do gateway (Arquitetura 2)
func NewProcessorGateway(defaultURL, fallbackURL string, redisCache *cache.RedisCache) *ProcessorGateway {
	return &ProcessorGateway{
		defaultURL:  defaultURL,
		fallbackURL: fallbackURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		redisCache: redisCache,
	}
}

// DecideProcessor escolhe qual processor usar baseado no cache Redis (Arquitetura 2)
func (pg *ProcessorGateway) DecideProcessor() (*ProcessorInfo, error) {
	log.Printf("🚀 Arquitetura 2: Iniciando decisão de processor via Redis Cache...")
	
	// 1. Primeiro, tentar obter do cache Redis
	cachedGateway, err := pg.redisCache.GetAvailableGateway()
	if err != nil {
		log.Printf("⚠️ Erro ao consultar cache Redis: %v", err)
		// Fallback para verificação direta se cache falhar
		return pg.decideFallbackWithoutCache()
	}
	
	// 2. Se há gateway no cache, usar ele
	if cachedGateway != nil && cachedGateway.IsAvailable {
		log.Printf("📋 Cache hit: usando processor %s (%s) do cache", 
			cachedGateway.Name, cachedGateway.URL)
		
		return &ProcessorInfo{
			URL:       cachedGateway.URL,
			Name:      cachedGateway.Name,
			IsDefault: cachedGateway.IsDefault,
		}, nil
	}
	
	// 3. Se não há gateway no cache, verificar diretamente (fallback)
	log.Printf("🔍 Cache miss: nenhum gateway disponível no cache, verificando diretamente...")
	return pg.decideFallbackWithoutCache()
}

// decideFallbackWithoutCache é usado quando o cache não está disponível (fallback da Arquitetura 2)
func (pg *ProcessorGateway) decideFallbackWithoutCache() (*ProcessorInfo, error) {
	log.Printf("⚠️ Fallback: verificando processors diretamente sem cache...")
	
	// Verificar Default Processor primeiro
	if pg.isProcessorUp(pg.defaultURL) {
		log.Printf("✅ Default Processor está UP (verificação direta): %s", pg.defaultURL)
		return &ProcessorInfo{
			URL:       pg.defaultURL,
			Name:      "default",
			IsDefault: true,
		}, nil
	}
	
	log.Printf("❌ Default Processor está DOWN, verificando Fallback...")
	
	// Se Default falhou, verificar Fallback
	if pg.isProcessorUp(pg.fallbackURL) {
		log.Printf("✅ Fallback Processor está UP (verificação direta): %s", pg.fallbackURL)
		return &ProcessorInfo{
			URL:       pg.fallbackURL,
			Name:      "fallback",
			IsDefault: false,
		}, nil
	}
	
	// Se ambos falharam
	log.Printf("❌ ERRO: Todos os processors estão DOWN (verificação direta)!")
	return nil, fmt.Errorf("nenhum payment processor está disponível")
}

// isProcessorUp verifica se um processor está funcionando
func (pg *ProcessorGateway) isProcessorUp(url string) bool {
	healthURL := fmt.Sprintf("%s/payments/service-health", url)
	
	resp, err := pg.httpClient.Get(healthURL)
	if err != nil {
		log.Printf("❌ Erro ao verificar health de %s: %v", url, err)
		return false
	}
	defer resp.Body.Close()
	
	isUp := resp.StatusCode == http.StatusOK
	if isUp {
		log.Printf("✅ Health check OK para %s", url)
	} else {
		log.Printf("❌ Health check falhou para %s: status %d", url, resp.StatusCode)
	}
	
	return isUp
}

// GetProcessorStatus retorna o status atual dos processors do cache Redis (Arquitetura 2)
func (pg *ProcessorGateway) GetProcessorStatus() map[string]bool {
	log.Printf("📊 Arquitetura 2: Consultando status dos processors no Redis Cache...")
	
	if pg.redisCache == nil {
		log.Printf("⚠️ Redis cache não está disponível, retornando status padrão")
		return map[string]bool{
			"default":  false,
			"fallback": false,
		}
	}
	
	// Obter status do cache Redis
	status := pg.redisCache.GetAllProcessorStatus()
	
	log.Printf("📋 Status dos processors do cache: Default=%t, Fallback=%t", 
		status["default"], status["fallback"])
	
	return status
} 