package services

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

type CircuitBreakerConfig struct {
	FailureThreshold   int
	SuccessThreshold   int
	Timeout           time.Duration
	MaxRetries        int
	ResetTimeout      time.Duration
}

type CircuitBreaker struct {
	config       CircuitBreakerConfig
	state        CircuitState
	failureCount int
	successCount int
	lastFailTime time.Time
	mu           sync.RWMutex
	name         string
}

type ProcessorCircuitBreakers struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

func NewProcessorCircuitBreakers() *ProcessorCircuitBreakers {
	return &ProcessorCircuitBreakers{
		breakers: make(map[string]*CircuitBreaker),
	}
}

func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config:       config,
		state:        CircuitClosed,
		failureCount: 0,
		successCount: 0,
		name:         name,
	}
}

func (pcb *ProcessorCircuitBreakers) GetOrCreateBreaker(processorName string) *CircuitBreaker {
	pcb.mu.RLock()
	breaker, exists := pcb.breakers[processorName]
	pcb.mu.RUnlock()
	
	if exists {
		return breaker
	}
	
	pcb.mu.Lock()
	defer pcb.mu.Unlock()
	
	if breaker, exists := pcb.breakers[processorName]; exists {
		return breaker
	}
	
	config := CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:         15 * time.Second,
		MaxRetries:      3,
		ResetTimeout:    30 * time.Second,
	}
	
	breaker = NewCircuitBreaker(processorName, config)
	pcb.breakers[processorName] = breaker
	
	log.Printf("CIRCUIT_BREAKER_CREATED: processor=%s threshold=%d timeout=%v", 
		processorName, config.FailureThreshold, config.ResetTimeout)
	
	return breaker
}

func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailTime) > cb.config.ResetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			if cb.state == CircuitOpen && time.Since(cb.lastFailTime) > cb.config.ResetTimeout {
				cb.state = CircuitHalfOpen
				cb.successCount = 0
				log.Printf("CIRCUIT_BREAKER_HALF_OPEN: processor=%s after=%v", 
					cb.name, time.Since(cb.lastFailTime))
			}
			cb.mu.Unlock()
			cb.mu.RLock()
			return cb.state == CircuitHalfOpen
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	switch cb.state {
	case CircuitClosed:
		cb.failureCount = 0
	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = CircuitClosed
			cb.failureCount = 0
			cb.successCount = 0
			log.Printf("CIRCUIT_BREAKER_CLOSED: processor=%s successes=%d", 
				cb.name, cb.successCount)
		}
	}
	
	log.Printf("CIRCUIT_BREAKER_SUCCESS: processor=%s state=%s failures=%d successes=%d", 
		cb.name, cb.getStateName(), cb.failureCount, cb.successCount)
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.lastFailTime = time.Now()
	cb.successCount = 0
	
	switch cb.state {
	case CircuitClosed:
		cb.failureCount++
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = CircuitOpen
			log.Printf("CIRCUIT_BREAKER_OPENED: processor=%s failures=%d threshold=%d", 
				cb.name, cb.failureCount, cb.config.FailureThreshold)
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.failureCount++
		log.Printf("CIRCUIT_BREAKER_REOPENED: processor=%s from_half_open", cb.name)
	}
	
	log.Printf("CIRCUIT_BREAKER_FAILURE: processor=%s state=%s failures=%d", 
		cb.name, cb.getStateName(), cb.failureCount)
}

func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) getStateName() string {
	switch cb.state {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

func (cb *CircuitBreaker) GetStats() (CircuitState, int, int, time.Time) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state, cb.failureCount, cb.successCount, cb.lastFailTime
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	if !cb.CanExecute() {
		return fmt.Errorf("circuit breaker is open for %s", cb.name)
	}
	
	err := fn()
	
	if err != nil {
		cb.RecordFailure()
		return err
	}
	
	cb.RecordSuccess()
	return nil
}