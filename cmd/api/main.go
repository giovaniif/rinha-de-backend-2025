package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"rinha-de-backend-2025/internal/handler"
)

func main() {
	// Carregar arquivo .env
	if err := godotenv.Load("config.env"); err != nil {
		log.Printf("Aviso: Não foi possível carregar config.env: %v", err)
	}

	// Configurar as URLs dos Payment Processors via variáveis de ambiente
	defaultURL := os.Getenv("PAYMENT_PROCESSOR_URL_DEFAULT")
	log.Printf("Default URL: %s", defaultURL)
	fallbackURL := os.Getenv("PAYMENT_PROCESSOR_URL_FALLBACK")
	log.Printf("Fallback URL: %s", fallbackURL)

	if defaultURL == "" {
		defaultURL = "http://localhost:8001"
	}
	if fallbackURL == "" {
		fallbackURL = "http://localhost:8002"
	}

	// Inicializar o handler
	h := handler.New(defaultURL, fallbackURL)

	// Configurar rotas
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/payments", h.ProcessPayment)

	// Iniciar o servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "9999"
	}

	log.Printf("Servidor iniciando na porta %s", port)
	log.Printf("Payment Processor Default: %s", defaultURL)
	log.Printf("Payment Processor Fallback: %s", fallbackURL)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
} 