package usecase

import (
	"fmt"
	"log"
	"time"

	"rinha-de-backend-2025/internal/gateway"
	"rinha-de-backend-2025/internal/payment"
	"rinha-de-backend-2025/internal/repository"
)

// PaymentUseCase orquestra todo o fluxo de processamento de pagamentos
type PaymentUseCase struct {
	gateway        *gateway.ProcessorGateway
	paymentClient  *payment.Client
	paymentRepo    repository.PaymentRepository
}

// PaymentResult representa o resultado do processamento
type PaymentResult struct {
	Success         bool                     `json:"success"`
	Payment         *payment.PaymentResponse `json:"payment,omitempty"`
	ProcessorUsed   string                   `json:"processor_used"`
	ProcessingTime  time.Duration           `json:"processing_time"`
	Error           string                   `json:"error,omitempty"`
	SavedToDB       bool                     `json:"saved_to_db"`
}

// NewPaymentUseCase cria uma nova instância do use case
func NewPaymentUseCase(
	gateway *gateway.ProcessorGateway,
	paymentClient *payment.Client,
	paymentRepo repository.PaymentRepository,
) *PaymentUseCase {
	return &PaymentUseCase{
		gateway:       gateway,
		paymentClient: paymentClient,
		paymentRepo:   paymentRepo,
	}
}

// ProcessPayment executa todo o fluxo de processamento conforme Arquitetura 1
func (uc *PaymentUseCase) ProcessPayment(req payment.PaymentRequest) *PaymentResult {
	startTime := time.Now()
	result := &PaymentResult{}
	
	log.Printf("Iniciando processamento de pagamento: correlationId=%s, amount=%.2f", 
		req.CorrelationID, req.Amount)
	
	// 1. Decide Processor Gateway
	processorInfo, err := uc.gateway.DecideProcessor()
	if err != nil {
		log.Printf("ERRO: Falha na decisão do processor: %v", err)
		result.Success = false
		result.Error = fmt.Sprintf("Nenhum processor disponível: %v", err)
		result.ProcessingTime = time.Since(startTime)
		
		// Fail Safe: Salva tentativa mesmo com erro
		uc.failSafe(req, "none", err)
		return result
	}
	
	result.ProcessorUsed = processorInfo.Name
	log.Printf("Processor selecionado: %s (%s)", processorInfo.Name, processorInfo.URL)
	
	// 2. Process Payment
	paymentResp, err := uc.paymentClient.ProcessPaymentWithURL(processorInfo.URL, req)
	if err != nil {
		log.Printf("ERRO: Falha no processamento do pagamento: %v", err)
		result.Success = false
		result.Error = fmt.Sprintf("Erro no processamento: %v", err)
		result.ProcessingTime = time.Since(startTime)
		
		// Fail Safe: Salva tentativa com erro
		uc.failSafe(req, processorInfo.Name, err)
		return result
	}
	
	// 3. Success Path
	result.Success = true
	result.Payment = paymentResp
	result.ProcessingTime = time.Since(startTime)
	
	log.Printf("Pagamento processado com sucesso: ID=%s, CorrelationID=%s, Status=%s", 
		paymentResp.ID, paymentResp.CorrelationID, paymentResp.Status)
	
	// 4. Save Payment Info
	saved := uc.savePaymentInfo(req, paymentResp, processorInfo.Name)
	result.SavedToDB = saved
	
	return result
}

// savePaymentInfo salva as informações do pagamento no banco
func (uc *PaymentUseCase) savePaymentInfo(
	req payment.PaymentRequest, 
	resp *payment.PaymentResponse, 
	processorName string,
) bool {
	paymentRecord := &repository.Payment{
		PaymentID:       resp.ID,
		CorrelationID:   req.CorrelationID,
		PaymentProcessor: processorName,
		Amount:          req.Amount,
		Status:          resp.Status,
		Fee:             resp.Fee,
		ProcessedAt:     time.Now(),
		CreatedAt:       time.Now(),
	}
	
	if err := uc.paymentRepo.Save(paymentRecord); err != nil {
		log.Printf("ERRO: Falha ao salvar pagamento no banco: %v", err)
		return false
	}
	
	log.Printf("Pagamento salvo com sucesso no banco: ID=%s, CorrelationID=%s", 
		resp.ID, req.CorrelationID)
	return true
}

// failSafe implementa o mecanismo de fail safe da arquitetura
func (uc *PaymentUseCase) failSafe(
	req payment.PaymentRequest, 
	processorName string, 
	err error,
) {
	log.Printf("Executando Fail Safe para pagamento: CorrelationID=%s", req.CorrelationID)
	
	failRecord := &repository.Payment{
		PaymentID:       fmt.Sprintf("fail_%d", time.Now().UnixNano()),
		CorrelationID:   req.CorrelationID,
		PaymentProcessor: processorName,
		Amount:          req.Amount,
		Status:          "failed",
		Fee:             0,
		ErrorMessage:    err.Error(),
		ProcessedAt:     time.Now(),
		CreatedAt:       time.Now(),
	}
	
	if saveErr := uc.paymentRepo.Save(failRecord); saveErr != nil {
		log.Printf("ERRO CRÍTICO: Falha no Fail Safe: %v", saveErr)
	} else {
		log.Printf("Fail Safe executado com sucesso: ID=%s, CorrelationID=%s", 
			failRecord.PaymentID, req.CorrelationID)
	}
}

// GetPaymentHistory busca o histórico de pagamentos
func (uc *PaymentUseCase) GetPaymentHistory(limit int) ([]*repository.Payment, error) {
	return uc.paymentRepo.FindAll(limit)
}

// GetPaymentByCorrelationID busca um pagamento específico pelo CorrelationID
func (uc *PaymentUseCase) GetPaymentByCorrelationID(correlationID string) (*repository.Payment, error) {
	return uc.paymentRepo.FindByCorrelationID(correlationID)
}

// GetProcessorStats retorna estatísticas dos processors
func (uc *PaymentUseCase) GetProcessorStats() map[string]interface{} {
	stats := uc.paymentRepo.GetProcessorStats()
	status := uc.gateway.GetProcessorStatus()
	
	return map[string]interface{}{
		"processor_usage": stats,
		"processor_status": status,
		"timestamp": time.Now(),
	}
} 

// GetPaymentsSummary retorna resumo de pagamentos por processor no período especificado
func (uc *PaymentUseCase) GetPaymentsSummary(from, to time.Time) (*repository.PaymentSummary, error) {
	log.Printf("Buscando resumo de pagamentos de %s até %s", from.Format(time.RFC3339), to.Format(time.RFC3339))
	
	// Validar período
	if from.After(to) {
		return nil, fmt.Errorf("data inicial não pode ser posterior à data final")
	}
	
	// Verificar se o período não é muito longo (máximo 1 ano)
	if to.Sub(from) > 365*24*time.Hour {
		return nil, fmt.Errorf("período máximo permitido é de 1 ano")
	}
	
	summary, err := uc.paymentRepo.GetPaymentsSummary(from, to)
	if err != nil {
		log.Printf("Erro ao buscar resumo: %v", err)
		return nil, fmt.Errorf("falha ao gerar resumo: %v", err)
	}
	
	return summary, nil
} 