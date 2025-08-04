package models

import "time"

type GatewaySelectionProfile struct {
	CorrelationID       string
	StartTime          time.Time
	RedisLookupTime    time.Duration
	HistoryLookupTime  time.Duration
	GracePeriodTime    time.Duration
	TotalTime          time.Duration
	SelectedGateway    string
	SelectionMethod    string
	RedisHits          int
	HistoryChecked     bool
	GracePeriodUsed    bool
}

type HealthCheckProfile struct {
	GatewayName       string
	StartTime         time.Time
	RequestTime       time.Duration
	RedisUpdateTime   time.Duration
	LockWaitTime      time.Duration
	TotalTime         time.Duration
	Success           bool
	StatusCode        int
}

type PaymentProfile struct {
	CorrelationID         string
	StartTime            time.Time
	GatewaySelectionTime time.Duration
	JSONSerializationTime time.Duration
	HTTPRequestTime      time.Duration
	HTTPWaitingTime      time.Duration
	JSONDeserializationTime time.Duration
	QueueTime            time.Duration
	TotalTime            time.Duration
	AttemptNumber        int
	Success              bool
	StatusCode           int
	PaymentProcessor     string
	ErrorType            string
}