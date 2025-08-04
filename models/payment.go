package models

import "time"

type PaymentRequest struct {
	CorrelationID string  `json:"correlationId" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
}

type PaymentProcessorRequest struct {
	CorrelationID string    `json:"correlationId"`
	Amount        float64   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
}

type HealthResponse struct {
	Failing           bool `json:"failing"`
	MinResponseTime   int  `json:"minResponseTime"`
}

type PaymentsSummaryResponse struct {
	Default  ProcessorSummary `json:"default"`
	Fallback ProcessorSummary `json:"fallback"`
}

type ProcessorSummary struct {
	TotalRequests int64   `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

type PaymentJob struct {
	PaymentRequest PaymentProcessorRequest
	ProcessorType  string
	Attempts       int
	MaxAttempts    int
}