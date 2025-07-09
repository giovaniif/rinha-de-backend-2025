package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"rinha-de-backend-2025/internal/payment"
)

type Handler struct {
	paymentClient *payment.Client
	defaultURL    string
	fallbackURL   string
}

func New(defaultURL, fallbackURL string) *Handler {
	return &Handler{
		paymentClient: payment.NewClient(defaultURL, fallbackURL),
		defaultURL:    defaultURL,
		fallbackURL:   fallbackURL,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "ok",
		"services": map[string]interface{}{
			"default":  h.checkService(h.defaultURL),
			"fallback": h.checkService(h.fallbackURL),
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

	// Validações básicas
	if req.Amount <= 0 {
		http.Error(w, "Valor deve ser maior que zero", http.StatusBadRequest)
		return
	}
	if req.Currency == "" {
		http.Error(w, "Moeda é obrigatória", http.StatusBadRequest)
		return
	}

	// Processar pagamento
	resp, err := h.paymentClient.ProcessPayment(req)
	if err != nil {
		fmt.Printf("Erro ao processar pagamento: %v\n", err)
		http.Error(w, "Erro ao processar pagamento", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) checkService(url string) string {
	if err := h.paymentClient.HealthCheck(url); err != nil {
		return "erro"
	}
	return "ok"
} 