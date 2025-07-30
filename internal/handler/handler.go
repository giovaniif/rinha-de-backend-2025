package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"rinha-de-backend-2025/internal/gateway"
	"rinha-de-backend-2025/internal/payment"
	"rinha-de-backend-2025/internal/usecase"
)

type Handler struct {
	paymentUseCase *usecase.PaymentUseCase
	gateway        *gateway.ProcessorGateway
	defaultURL     string
	fallbackURL    string
}

func New(defaultURL, fallbackURL string, paymentUseCase *usecase.PaymentUseCase) *Handler {
	return &Handler{
		paymentUseCase: paymentUseCase,
		gateway:        gateway.NewProcessorGateway(defaultURL, fallbackURL),
		defaultURL:     defaultURL,
		fallbackURL:    fallbackURL,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	processorStatus := h.gateway.GetProcessorStatus()
	
	status := map[string]interface{}{
		"status": "ok",
		"architecture": "Arquitetura 1 - Rinha Backend 2025",
		"components": map[string]interface{}{
			"processor_gateway": "online",
			"payment_usecase":   "online",
			"database":          "online",
		},
		"processors": processorStatus,
		"endpoints": []string{
			"POST /payments - Processar pagamento",
			"GET /payments/history - Histórico de pagamentos",
			"GET /payments/stats - Estatísticas dos processors",
			"GET /health - Status dos serviços",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *Handler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req payment.PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// Validações do payload da Rinha de Backend 2025
	if req.CorrelationID == "" {
		http.Error(w, "correlationId é obrigatório", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "amount deve ser maior que zero", http.StatusBadRequest)
		return
	}

	// Processar pagamento usando o Use Case da Arquitetura 1
	result := h.paymentUseCase.ProcessPayment(req)

	// Retornar resultado
	if result.Success {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":         true,
			"payment":         result.Payment,
			"processor_used":  result.ProcessorUsed,
			"processing_time": result.ProcessingTime.String(),
			"saved_to_db":     result.SavedToDB,
		})
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":         false,
			"error":           result.Error,
			"processor_used":  result.ProcessorUsed,
			"processing_time": result.ProcessingTime.String(),
		})
	}
}

func (h *Handler) PaymentHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Obter limite da query string (padrão: 10)
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	payments, err := h.paymentUseCase.GetPaymentHistory(limit)
	if err != nil {
		fmt.Printf("Erro ao buscar histórico: %v\n", err)
		http.Error(w, "Erro ao buscar histórico", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payments": payments,
		"total":    len(payments),
		"limit":    limit,
	})
}

func (h *Handler) ProcessorStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	stats := h.paymentUseCase.GetProcessorStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) PaymentsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Extrair parâmetros de query
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	// Validar se os parâmetros foram fornecidos
	if fromStr == "" || toStr == "" {
		http.Error(w, "Parâmetros 'from' e 'to' são obrigatórios (formato: RFC3339)", http.StatusBadRequest)
		return
	}

	// Parsear as datas (formato RFC3339: 2020-07-10T12:34:56.000Z)
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		http.Error(w, "Formato inválido para 'from'. Use formato RFC3339: 2020-07-10T12:34:56.000Z", http.StatusBadRequest)
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		http.Error(w, "Formato inválido para 'to'. Use formato RFC3339: 2020-07-10T12:34:56.000Z", http.StatusBadRequest)
		return
	}

	// Buscar resumo usando o UseCase
	summary, err := h.paymentUseCase.GetPaymentsSummary(from, to)
	if err != nil {
		fmt.Printf("Erro ao buscar resumo: %v\n", err)
		http.Error(w, fmt.Sprintf("Erro ao gerar resumo: %v", err), http.StatusInternalServerError)
		return
	}

	// Retornar resultado
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
} 