package gateway

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// ProcessorGateway implementa a lógica de decisão entre Default e Fallback Processors
type ProcessorGateway struct {
	defaultURL  string
	fallbackURL string
	httpClient  *http.Client
}

// ProcessorInfo contém informações sobre qual processor usar
type ProcessorInfo struct {
	URL       string
	Name      string
	IsDefault bool
}

// NewProcessorGateway cria uma nova instância do gateway
func NewProcessorGateway(defaultURL, fallbackURL string) *ProcessorGateway {
	return &ProcessorGateway{
		defaultURL:  defaultURL,
		fallbackURL: fallbackURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // Timeout mais rápido para health check
		},
	}
}

// DecideProcessor implementa a lógica de decisão conforme a Arquitetura 1:
// 1. Verifica se o Default Processor está UP
// 2. Se estiver UP, usa o Default
// 3. Se não estiver UP, verifica o Fallback
// 4. Se Fallback estiver UP, usa o Fallback
// 5. Se nenhum estiver UP, retorna erro
func (pg *ProcessorGateway) DecideProcessor() (*ProcessorInfo, error) {
	log.Printf("Iniciando decisão de processor...")
	
	// Primeiro, tenta o Default Processor
	if pg.isProcessorUp(pg.defaultURL) {
		log.Printf("Default Processor está UP, usando: %s", pg.defaultURL)
		return &ProcessorInfo{
			URL:       pg.defaultURL,
			Name:      "default",
			IsDefault: true,
		}, nil
	}
	
	log.Printf("Default Processor está DOWN, verificando Fallback...")
	
	// Se Default falhou, tenta Fallback
	if pg.isProcessorUp(pg.fallbackURL) {
		log.Printf("Fallback Processor está UP, usando: %s", pg.fallbackURL)
		return &ProcessorInfo{
			URL:       pg.fallbackURL,
			Name:      "fallback", 
			IsDefault: false,
		}, nil
	}
	
	// Se ambos falharam
	log.Printf("ERRO: Todos os processors estão DOWN!")
	return nil, fmt.Errorf("nenhum payment processor está disponível")
}

// isProcessorUp verifica se um processor está funcionando
func (pg *ProcessorGateway) isProcessorUp(url string) bool {
	healthURL := url + "/payments/service-health"
	
	resp, err := pg.httpClient.Get(healthURL)
	if err != nil {
		log.Printf("Erro ao verificar health de %s: %v", url, err)
		return false
	}
	defer resp.Body.Close()
	
	isUp := resp.StatusCode == http.StatusOK
	log.Printf("Health check %s: %d (UP: %v)", url, resp.StatusCode, isUp)
	
	return isUp
}

// GetProcessorStatus retorna o status de ambos os processors
func (pg *ProcessorGateway) GetProcessorStatus() map[string]bool {
	return map[string]bool{
		"default":  pg.isProcessorUp(pg.defaultURL),
		"fallback": pg.isProcessorUp(pg.fallbackURL),
	}
} 