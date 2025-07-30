package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"rinha-de-backend-2025/internal/cache"
	"rinha-de-backend-2025/internal/gateway"
	"rinha-de-backend-2025/internal/handler"
	"rinha-de-backend-2025/internal/payment"
	"rinha-de-backend-2025/internal/repository"
	"rinha-de-backend-2025/internal/usecase"
)

func main() {
	log.Printf("=== Rinha de Backend 2025 - Arquitetura 2 ===")

	// 1. Carregar variáveis de ambiente
	port := getEnvOrDefault("PORT", "8080")
	defaultProcessorURL := getEnvOrDefault("PAYMENT_PROCESSOR_URL_DEFAULT", "http://localhost:8001")
	fallbackProcessorURL := getEnvOrDefault("PAYMENT_PROCESSOR_URL_FALLBACK", "http://localhost:8002")
	databaseURL := getEnvOrDefault("DATABASE_URL", "postgres://user:password@localhost/dbname?sslmode=disable")
	redisURL := getEnvOrDefault("REDIS_URL", "redis://localhost:6379") // Arquitetura 2

	log.Printf("Default Processor URL: %s", maskPassword(defaultProcessorURL))
	log.Printf("Fallback Processor URL: %s", maskPassword(fallbackProcessorURL))
	log.Printf("Database URL: %s", maskPassword(databaseURL))
	log.Printf("Redis URL: %s", maskPassword(redisURL)) // Arquitetura 2

	// 2. Inicializar Redis Cache (Arquitetura 2)
	log.Printf("Inicializando Redis Cache...")
	redisCache, err := cache.NewRedisCache(redisURL)
	if err != nil {
		log.Fatalf("Erro ao inicializar Redis Cache: %v", err)
	}
	defer redisCache.Close()

	// 3. Inicializar banco de dados PostgreSQL
	log.Printf("Inicializando banco de dados...")
	db, err := repository.InitDatabase(databaseURL)
	if err != nil {
		log.Fatalf("Erro ao inicializar banco: %v", err)
	}
	defer db.Close()

	// 4. Configurar componentes da Arquitetura 2
	log.Printf("Configurando componentes da Arquitetura 2...")
	
	// Gateway com Redis Cache
	processorGateway := gateway.NewProcessorGateway(defaultProcessorURL, fallbackProcessorURL, redisCache)
	
	// Payment Client
	paymentClient := payment.NewClient(defaultProcessorURL, fallbackProcessorURL)
	
	// Payment Repository
	paymentRepo := repository.NewPostgreSQLPaymentRepository(db)
	
	// Payment Use Case
	paymentUseCase := usecase.NewPaymentUseCase(processorGateway, paymentClient, paymentRepo)
	
	// Gateway Instance que roda em paralelo (Arquitetura 2)
	gatewayInstance := gateway.NewGatewayInstance(defaultProcessorURL, fallbackProcessorURL, redisCache)

	// 5. Iniciar Gateway Instance em background (Arquitetura 2)
	log.Printf("Iniciando Gateway Instance em paralelo...")
	gatewayInstance.Start()
	defer gatewayInstance.Stop()

	// 6. Configurar handlers
	h := handler.New(defaultProcessorURL, fallbackProcessorURL, paymentUseCase)

	// 7. Configurar rotas
	log.Printf("Configurando rotas...")
	mux := http.NewServeMux()
	mux.HandleFunc("/payments", h.ProcessPayment)
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/payments/history", h.PaymentHistory)
	mux.HandleFunc("/payments/stats", h.ProcessorStats)
	mux.HandleFunc("/payments-summary", h.PaymentsSummary)

	// 8. Configurar graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		log.Printf("🛑 Sinal de parada recebido, encerrando aplicação...")
		gatewayInstance.Stop()
		redisCache.Close()
		db.Close()
		os.Exit(0)
	}()

	// 9. Iniciar o servidor
	log.Printf("=== Servidor Arquitetura 2 iniciando ===")
	log.Printf("Porta: %s", port)
	log.Printf("Endpoint principal: POST /payments")
	log.Printf("Health check: GET /health")
	log.Printf("Histórico: GET /payments/history")
	log.Printf("Estatísticas: GET /payments/stats")
	log.Printf("Resumo: GET /payments-summary")
	log.Printf("Redis Cache: ✅ Ativo")
	log.Printf("Gateway Instance: ✅ Rodando em paralelo (5s intervals)")
	log.Printf("=====================================")

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}

// getEnvOrDefault retorna uma variável de ambiente ou valor padrão
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// maskPassword oculta senhas em URLs para logs seguros
func maskPassword(url string) string {
	if strings.Contains(url, "://") && strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 {
			schemeParts := strings.Split(parts[0], "://")
			if len(schemeParts) == 2 {
				return schemeParts[0] + "://***@" + parts[1]
			}
		}
	}
	return url
} 