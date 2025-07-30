package repository

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Payment representa um registro na tabela payments
type Payment struct {
	ID               int       `json:"id" db:"id"`
	PaymentID        string    `json:"payment_id" db:"payment_id"`
	CorrelationID    string    `json:"correlation_id" db:"correlation_id"`
	PaymentProcessor string    `json:"payment_processor" db:"payment_processor"`
	Amount           float64   `json:"amount" db:"amount"`
	Status           string    `json:"status" db:"status"`
	Fee              float64   `json:"fee" db:"fee"`
	ErrorMessage     string    `json:"error_message,omitempty" db:"error_message"`
	ProcessedAt      time.Time `json:"processed_at" db:"processed_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// ProcessorSummary representa estatísticas de um processor específico
type ProcessorSummary struct {
	TotalRequests int64   `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

// PaymentSummary representa o resumo completo de pagamentos
type PaymentSummary struct {
	Default  ProcessorSummary `json:"default"`
	Fallback ProcessorSummary `json:"fallback"`
}

// PaymentRepository interface para operações de pagamento
type PaymentRepository interface {
	Save(payment *Payment) error
	FindByID(paymentID string) (*Payment, error)
	FindByCorrelationID(correlationID string) (*Payment, error)
	FindAll(limit int) ([]*Payment, error)
	GetProcessorStats() map[string]int
	GetPaymentsSummary(from, to time.Time) (*PaymentSummary, error)
}

// PostgreSQLPaymentRepository implementação PostgreSQL
type PostgreSQLPaymentRepository struct {
	db *sql.DB
}

// NewPostgreSQLPaymentRepository cria uma nova instância do repositório
func NewPostgreSQLPaymentRepository(db *sql.DB) PaymentRepository {
	return &PostgreSQLPaymentRepository{db: db}
}

// Save salva um pagamento no banco de dados
func (r *PostgreSQLPaymentRepository) Save(payment *Payment) error {
	query := `
		INSERT INTO payments (
			payment_id, correlation_id, payment_processor, amount, 
			status, fee, error_message, processed_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`
	
	err := r.db.QueryRow(
		query,
		payment.PaymentID,
		payment.CorrelationID,
		payment.PaymentProcessor,
		payment.Amount,
		payment.Status,
		payment.Fee,
		payment.ErrorMessage,
		payment.ProcessedAt,
		payment.CreatedAt,
	).Scan(&payment.ID)
	
	if err != nil {
		return fmt.Errorf("erro ao salvar pagamento: %v", err)
	}
	
	log.Printf("Pagamento salvo no banco: ID=%d, PaymentID=%s, CorrelationID=%s", 
		payment.ID, payment.PaymentID, payment.CorrelationID)
	return nil
}

// FindByID busca um pagamento pelo PaymentID
func (r *PostgreSQLPaymentRepository) FindByID(paymentID string) (*Payment, error) {
	query := `
		SELECT id, payment_id, correlation_id, payment_processor, amount,
			   status, fee, error_message, processed_at, created_at
		FROM payments 
		WHERE payment_id = $1`
	
	payment := &Payment{}
	err := r.db.QueryRow(query, paymentID).Scan(
		&payment.ID,
		&payment.PaymentID,
		&payment.CorrelationID,
		&payment.PaymentProcessor,
		&payment.Amount,
		&payment.Status,
		&payment.Fee,
		&payment.ErrorMessage,
		&payment.ProcessedAt,
		&payment.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pagamento não encontrado: %s", paymentID)
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar pagamento: %v", err)
	}
	
	return payment, nil
}

// FindByCorrelationID busca um pagamento pelo CorrelationID
func (r *PostgreSQLPaymentRepository) FindByCorrelationID(correlationID string) (*Payment, error) {
	query := `
		SELECT id, payment_id, correlation_id, payment_processor, amount,
			   status, fee, error_message, processed_at, created_at
		FROM payments 
		WHERE correlation_id = $1`
	
	payment := &Payment{}
	err := r.db.QueryRow(query, correlationID).Scan(
		&payment.ID,
		&payment.PaymentID,
		&payment.CorrelationID,
		&payment.PaymentProcessor,
		&payment.Amount,
		&payment.Status,
		&payment.Fee,
		&payment.ErrorMessage,
		&payment.ProcessedAt,
		&payment.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pagamento não encontrado: %s", correlationID)
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar pagamento: %v", err)
	}
	
	return payment, nil
}

// FindAll busca todos os pagamentos com limite
func (r *PostgreSQLPaymentRepository) FindAll(limit int) ([]*Payment, error) {
	query := `
		SELECT id, payment_id, correlation_id, payment_processor, amount,
			   status, fee, error_message, processed_at, created_at
		FROM payments 
		ORDER BY created_at DESC 
		LIMIT $1`
	
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar pagamentos: %v", err)
	}
	defer rows.Close()
	
	var payments []*Payment
	for rows.Next() {
		payment := &Payment{}
		err := rows.Scan(
			&payment.ID,
			&payment.PaymentID,
			&payment.CorrelationID,
			&payment.PaymentProcessor,
			&payment.Amount,
			&payment.Status,
			&payment.Fee,
			&payment.ErrorMessage,
			&payment.ProcessedAt,
			&payment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao escanear pagamento: %v", err)
		}
		payments = append(payments, payment)
	}
	
	return payments, nil
}

// GetProcessorStats retorna estatísticas de uso dos processors
func (r *PostgreSQLPaymentRepository) GetProcessorStats() map[string]int {
	query := `
		SELECT payment_processor, COUNT(*) as count
		FROM payments 
		GROUP BY payment_processor`
	
	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("Erro ao buscar estatísticas: %v", err)
		return make(map[string]int)
	}
	defer rows.Close()
	
	stats := make(map[string]int)
	for rows.Next() {
		var processor string
		var count int
		
		if err := rows.Scan(&processor, &count); err != nil {
			log.Printf("Erro ao escanear estatística: %v", err)
			continue
		}
		
		stats[processor] = count
	}
	
	return stats
}

// GetPaymentsSummary retorna estatísticas de pagamentos por processor em um período
func (r *PostgreSQLPaymentRepository) GetPaymentsSummary(from, to time.Time) (*PaymentSummary, error) {
	query := `
		SELECT 
			payment_processor,
			COUNT(*) as total_requests,
			COALESCE(SUM(amount), 0) as total_amount
		FROM payments 
		WHERE created_at >= $1 AND created_at <= $2
			AND status != 'failed'
		GROUP BY payment_processor`
	
	rows, err := r.db.Query(query, from, to)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar resumo de pagamentos: %v", err)
	}
	defer rows.Close()
	
	summary := &PaymentSummary{
		Default:  ProcessorSummary{TotalRequests: 0, TotalAmount: 0},
		Fallback: ProcessorSummary{TotalRequests: 0, TotalAmount: 0},
	}
	
	for rows.Next() {
		var processor string
		var totalRequests int64
		var totalAmount float64
		
		if err := rows.Scan(&processor, &totalRequests, &totalAmount); err != nil {
			log.Printf("Erro ao escanear resumo: %v", err)
			continue
		}
		
		switch processor {
		case "default":
			summary.Default.TotalRequests = totalRequests
			summary.Default.TotalAmount = totalAmount
		case "fallback":
			summary.Fallback.TotalRequests = totalRequests
			summary.Fallback.TotalAmount = totalAmount
		}
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao iterar resultados: %v", err)
	}
	
	log.Printf("Resumo gerado: Default=%d req/%.2f, Fallback=%d req/%.2f", 
		summary.Default.TotalRequests, summary.Default.TotalAmount,
		summary.Fallback.TotalRequests, summary.Fallback.TotalAmount)
	
	return summary, nil
}

// InitDatabase inicializa o banco de dados e cria as tabelas necessárias
func InitDatabase(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar com banco: %v", err)
	}
	
	// Testar conexão
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao pingar banco: %v", err)
	}
	
	// Criar tabela se não existir
	if err = createPaymentsTable(db); err != nil {
		return nil, fmt.Errorf("erro ao criar tabela: %v", err)
	}
	
	log.Printf("Banco de dados inicializado com sucesso")
	return db, nil
}

// createPaymentsTable cria a tabela payments conforme a Arquitetura 1
func createPaymentsTable(db *sql.DB) error {
	// Criação da tabela (protegida contra execução simultânea)
	tableQuery := `
		CREATE TABLE IF NOT EXISTS payments (
			id SERIAL PRIMARY KEY,
			payment_id VARCHAR(255) UNIQUE NOT NULL,
			correlation_id VARCHAR(255) NOT NULL,
			payment_processor VARCHAR(50) NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			status VARCHAR(20) NOT NULL,
			fee DECIMAL(10,2) DEFAULT 0,
			error_message TEXT,
			processed_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
	
	_, err := db.Exec(tableQuery)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela payments: %v", err)
	}
	
	// Criação dos índices (um por vez para evitar conflitos)
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_payments_processor ON payments(payment_processor);",
		"CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);", 
		"CREATE INDEX IF NOT EXISTS idx_payments_correlation_id ON payments(correlation_id);",
		"CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at);",
	}
	
	for _, indexQuery := range indexes {
		_, err = db.Exec(indexQuery)
		if err != nil {
			// Log do erro mas não falha - índices são opcionais
			log.Printf("Aviso: erro ao criar índice: %v", err)
		}
	}
	
	log.Printf("Tabela payments criada/verificada com sucesso")
	return nil
} 