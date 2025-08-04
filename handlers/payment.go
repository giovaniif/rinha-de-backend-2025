package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"payment-processor/models"
	"payment-processor/services"
)

type PaymentHandler struct {
	paymentService *services.PaymentService
}

func NewPaymentHandler(paymentService *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	var req models.PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR_PARSE_JSON: %s | IP: %s | Body: %s", err.Error(), c.ClientIP(), c.Request.Body)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("PAYMENT_REQUEST: correlationId=%s amount=%.2f", req.CorrelationID, req.Amount)

	err := h.paymentService.ProcessPaymentRequest(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "no gateway available" {
			log.Printf("ERROR_NO_GATEWAY: correlationId=%s amount=%.2f", req.CorrelationID, req.Amount)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service unavailable"})
			return
		}
		if err.Error() == "payment queue is full" {
			log.Printf("ERROR_QUEUE_FULL: correlationId=%s amount=%.2f", req.CorrelationID, req.Amount)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "queue full"})
			return
		}
		log.Printf("ERROR_INTERNAL: correlationId=%s amount=%.2f error=%s", req.CorrelationID, req.Amount, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("PAYMENT_ACCEPTED: correlationId=%s amount=%.2f", req.CorrelationID, req.Amount)
	c.JSON(http.StatusOK, gin.H{})
}

func (h *PaymentHandler) GetPaymentsSummary(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to parameters are required"})
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date format"})
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date format"})
		return
	}

	summary := h.paymentService.GetPaymentsSummary(from, to)
	c.JSON(http.StatusOK, summary)
}