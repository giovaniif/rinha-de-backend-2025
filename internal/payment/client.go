package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient  *http.Client
	defaultURL  string
	fallbackURL string
}

type PaymentRequest struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
}

type PaymentResponse struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	Fee         float64 `json:"fee"`
	ProcessedAt string  `json:"processed_at"`
}

func NewClient(defaultURL, fallbackURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		defaultURL:  defaultURL,
		fallbackURL: fallbackURL,
	}
}

func (c *Client) ProcessPayment(req PaymentRequest) (*PaymentResponse, error) {
	// Tentar primeiro com o serviço default
	resp, err := c.sendPaymentRequest(c.defaultURL, req)
	if err == nil {
		return resp, nil
	}

	// Se falhar, tentar com o fallback
	fmt.Printf("Erro no serviço default: %v. Tentando fallback...\n", err)
	return c.sendPaymentRequest(c.fallbackURL, req)
}

func (c *Client) sendPaymentRequest(url string, req PaymentRequest) (*PaymentResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", url+"/payments", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("erro do servidor: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro na resposta: status %d, body: %s", resp.StatusCode, string(body))
	}

	var paymentResp PaymentResponse
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		return nil, fmt.Errorf("erro ao deserializar resposta: %v", err)
	}

	return &paymentResp, nil
}

func (c *Client) HealthCheck(url string) error {
	resp, err := c.httpClient.Get(url + "/payments/service-health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check falhou: status %d", resp.StatusCode)
	}

	return nil
} 