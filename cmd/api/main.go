package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"rinha-de-backend-2025/internal/gateway"
	"rinha-de-backend-2025/internal/handler"
	"rinha-de-backend-2025/internal/payment"
	"rinha-de-backend-2025/internal/repository"
	"rinha-de-backend-2025/internal/usecase"
)

func main() {
	log.Printf("=== Rinha de Backend 2025 - Arquitetura 1 ===")
	
	// Carregar arquivo .env
	if err := godotenv.Load("config.env"); err != nil {
		log.Printf("Aviso: Não foi possível carregar config.env: %v", err)
	}

	// Configurar as URLs dos Payment Processors via variáveis de ambiente
	defaultURL := getEnvOrDefault("PAYMENT_PROCESSOR_URL_DEFAULT", "http://localhost:8001")
	fallbackURL := getEnvOrDefault("PAYMENT_PROCESSOR_URL_FALLBACK", "http://localhost:8002")
	databaseURL := getEnvOrDefault("DATABASE_URL", "postgres://user:password@localhost:5432/rinha_payments?sslmode=disable")
	
	log.Printf("Default Processor URL: %s", defaultURL)
	log.Printf("Fallback Processor URL: %s", fallbackURL)
	log.Printf("Database URL: %s", maskPassword(databaseURL))

	// 1. Inicializar banco de dados
	log.Printf("Inicializando banco de dados...")
	db, err := repository.InitDatabase(databaseURL)
	if err != nil {
		log.Fatalf("Erro ao inicializar banco: %v", err)
	}
	defer db.Close()

	// 2. Criar instâncias dos componentes da Arquitetura 1
	log.Printf("Configurando componentes da Arquitetura 1...")
	
	// Gateway de decisão de processors
	processorGateway := gateway.NewProcessorGateway(defaultURL, fallbackURL)
	
	// Cliente de pagamento
	paymentClient := payment.NewClient(defaultURL, fallbackURL)
	
	// Repositório de pagamentos
	paymentRepo := repository.NewPostgreSQLPaymentRepository(db)
	
	// Use Case principal
	paymentUseCase := usecase.NewPaymentUseCase(processorGateway, paymentClient, paymentRepo)
	
	// Handler HTTP
	h := handler.New(defaultURL, fallbackURL, paymentUseCase)

	// 3. Configurar rotas
	log.Printf("Configurando rotas...")
	mux := http.NewServeMux()
	
	// Rota principal da Arquitetura 1
	mux.HandleFunc("/payments", h.ProcessPayment)
	
	// Rotas auxiliares
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/payments/history", h.PaymentHistory)
	mux.HandleFunc("/payments/stats", h.ProcessorStats)
	mux.HandleFunc("/payments-summary", h.PaymentsSummary)

	// 4. Iniciar o servidor
	port := getEnvOrDefault("PORT", "9999")
	
	log.Printf("=== Servidor Arquitetura 1 iniciando ===")
	log.Printf("Porta: %s", port)
	log.Printf("Endpoint principal: POST /payments")
	log.Printf("Health check: GET /health")
	log.Printf("Histórico: GET /payments/history")
	log.Printf("Estatísticas: GET /payments/stats")
	log.Printf("=====================================")

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func maskPassword(url string) string {
	// Simples mascaramento da senha na URL para logs
	if len(url) > 20 {
		return url[:20] + "***"
	}
	return "***"
} 