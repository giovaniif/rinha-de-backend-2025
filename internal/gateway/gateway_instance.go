package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"rinha-de-backend-2025/internal/cache"
)

// GatewayInstance representa uma instância do gateway que roda em paralelo
type GatewayInstance struct {
	defaultURL   string
	fallbackURL  string
	redisCache   *cache.RedisCache
	httpClient   *http.Client
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	isRunning    bool
	mu           sync.RWMutex
}

// NewGatewayInstance cria uma nova instância do gateway
func NewGatewayInstance(defaultURL, fallbackURL string, redisCache *cache.RedisCache) *GatewayInstance {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &GatewayInstance{
		defaultURL:  defaultURL,
		fallbackURL: fallbackURL,
		redisCache:  redisCache,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start inicia o Gateway Instance em background
func (gi *GatewayInstance) Start() {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	
	if gi.isRunning {
		log.Printf("⚠️ Gateway Instance já está rodando")
		return
	}
	
	gi.isRunning = true
	
	log.Printf("🚀 Iniciando Gateway Instance...")
	log.Printf("   - Default Processor: %s", gi.defaultURL)
	log.Printf("   - Fallback Processor: %s", gi.fallbackURL)
	log.Printf("   - Health Check Interval: 5s")
	
	// Fazer um health check inicial imediato
	go gi.performInitialHealthCheck()
	
	// Iniciar loop de health checks a cada 5 segundos
	gi.wg.Add(1)
	go gi.healthCheckLoop()
}

// Stop para o Gateway Instance
func (gi *GatewayInstance) Stop() {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	
	if !gi.isRunning {
		return
	}
	
	log.Printf("🛑 Parando Gateway Instance...")
	gi.cancel()
	gi.wg.Wait()
	gi.isRunning = false
	log.Printf("✅ Gateway Instance parado")
}

// IsRunning retorna se o Gateway Instance está rodando
func (gi *GatewayInstance) IsRunning() bool {
	gi.mu.RLock()
	defer gi.mu.RUnlock()
	return gi.isRunning
}

// performInitialHealthCheck faz uma verificação inicial dos processors
func (gi *GatewayInstance) performInitialHealthCheck() {
	log.Printf("🔍 Executando health check inicial...")
	
	// Verificar Default Processor
	defaultUp := gi.checkProcessorHealth(gi.defaultURL)
	gi.redisCache.SetProcessorStatus("default", defaultUp)
	
	// Verificar Fallback Processor
	fallbackUp := gi.checkProcessorHealth(gi.fallbackURL)
	gi.redisCache.SetProcessorStatus("fallback", fallbackUp)
	
	// Atualizar cache com o melhor processor disponível
	gi.updateAvailableGateway(defaultUp, fallbackUp)
	
	log.Printf("✅ Health check inicial concluído: Default=%t, Fallback=%t", defaultUp, fallbackUp)
}

// healthCheckLoop executa health checks a cada 5 segundos
func (gi *GatewayInstance) healthCheckLoop() {
	defer gi.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-gi.ctx.Done():
			log.Printf("🔄 Health check loop encerrado")
			return
		case <-ticker.C:
			gi.performHealthCheck()
		}
	}
}

// performHealthCheck executa um ciclo completo de health check
func (gi *GatewayInstance) performHealthCheck() {
	log.Printf("🔄 Executando health check automático...")
	
	// Verificar Default Processor
	defaultUp := gi.checkProcessorHealth(gi.defaultURL)
	gi.redisCache.SetProcessorStatus("default", defaultUp)
	
	// Verificar Fallback Processor
	fallbackUp := gi.checkProcessorHealth(gi.fallbackURL)
	gi.redisCache.SetProcessorStatus("fallback", fallbackUp)
	
	// Atualizar cache com o melhor processor disponível
	gi.updateAvailableGateway(defaultUp, fallbackUp)
	
	log.Printf("🔍 Health check concluído: Default=%t, Fallback=%t", defaultUp, fallbackUp)
}

// checkProcessorHealth verifica se um processor específico está healthy
func (gi *GatewayInstance) checkProcessorHealth(url string) bool {
	healthURL := fmt.Sprintf("%s/payments/service-health", url)
	
	resp, err := gi.httpClient.Get(healthURL)
	if err != nil {
		log.Printf("❌ Health check falhou para %s: %v", url, err)
		return false
	}
	defer resp.Body.Close()
	
	isHealthy := resp.StatusCode == http.StatusOK
	if isHealthy {
		log.Printf("✅ Processor %s está healthy", url)
	} else {
		log.Printf("❌ Processor %s retornou status %d", url, resp.StatusCode)
	}
	
	return isHealthy
}

// updateAvailableGateway atualiza o cache com o melhor processor disponível
func (gi *GatewayInstance) updateAvailableGateway(defaultUp, fallbackUp bool) {
	// Buscar o gateway atual do cache
	currentGateway, _ := gi.redisCache.GetAvailableGateway()
	
	var newGateway *cache.ProcessorInfo
	
	// Priorizar Default Processor se estiver UP
	if defaultUp {
		newGateway = &cache.ProcessorInfo{
			URL:         gi.defaultURL,
			Name:        "default",
			IsDefault:   true,
			IsAvailable: true,
		}
	} else if fallbackUp {
		newGateway = &cache.ProcessorInfo{
			URL:         gi.fallbackURL,
			Name:        "fallback",
			IsDefault:   false,
			IsAvailable: true,
		}
	}
	
	// Se nenhum processor está UP, invalidar cache
	if newGateway == nil {
		if currentGateway != nil {
			log.Printf("⚠️ Todos os processors estão DOWN, invalidando cache")
			gi.redisCache.InvalidateGateway(currentGateway.Name)
		}
		return
	}
	
	// Se o gateway mudou ou não há gateway no cache, atualizar
	if currentGateway == nil || currentGateway.Name != newGateway.Name {
		if err := gi.redisCache.SetAvailableGateway(newGateway); err != nil {
			log.Printf("❌ Erro ao atualizar gateway no cache: %v", err)
		} else {
			log.Printf("🔄 Gateway atualizado no cache: %s (%s)", newGateway.Name, newGateway.URL)
		}
	}
} 