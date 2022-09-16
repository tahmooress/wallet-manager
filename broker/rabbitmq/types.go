package rabbitmq

import (
	"time"
)

type Config struct {
	Host              string
	Port              string
	ExchangeName      string
	ExchangeType      string
	Queue             string
	RouteKey          string
	addr              string
	consumerTag       string
	prefetchCount     int64
	prefetchSize      int64
	reconnectInterval time.Duration
	reInitInterval    time.Duration
}

type HealthState struct {
	Er      error
	Status  bool
	Message string
}
