package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"payment-processor/handlers"
	"payment-processor/services"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	gatewayService := services.NewGatewayService(redisClient)
	paymentService := services.NewPaymentService(redisClient, gatewayService)
	paymentHandler := handlers.NewPaymentHandler(paymentService)

	gatewayService.StartHealthChecker(ctx)
	paymentService.StartPaymentProcessor(ctx, 8)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		status := c.Writer.Status()
		
		if status >= 400 {
			log.Printf("HTTP_ERROR: method=%s path=%s status=%d duration=%v ip=%s", 
				c.Request.Method, c.Request.URL.Path, status, duration, c.ClientIP())
		} else {
			log.Printf("HTTP_OK: method=%s path=%s status=%d duration=%v", 
				c.Request.Method, c.Request.URL.Path, status, duration)
		}
	})

	router.POST("/payments", paymentHandler.ProcessPayment)
	router.GET("/payments-summary", paymentHandler.GetPaymentsSummary)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Println("Server started on :8080")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}