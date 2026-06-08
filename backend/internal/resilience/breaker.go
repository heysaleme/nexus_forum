package resilience

import (
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sony/gobreaker"
)

const (
	BreakerOAuth    = "oauth"
	BreakerMinIO    = "minio"
	BreakerExternal = "external_http"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker open")

	breakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state: 0=closed, 1=open, 2=half-open",
		},
		[]string{"name"},
	)

	breakerFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_failures_total",
			Help: "Total failures recorded by circuit breakers",
		},
		[]string{"name"},
	)

	registry = map[string]*gobreaker.CircuitBreaker{}
)

func init() {
	prometheus.MustRegister(breakerState, breakerFailures)
}

func stateToMetric(state gobreaker.State) float64 {
	switch state {
	case gobreaker.StateOpen:
		return 1
	case gobreaker.StateHalfOpen:
		return 2
	default:
		return 0
	}
}

// Get returns a named circuit breaker (created on first use).
func Get(name string) *gobreaker.CircuitBreaker {
	if cb, ok := registry[name]; ok {
		return cb
	}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 2,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			breakerState.WithLabelValues(name).Set(stateToMetric(to))
		},
	})
	breakerState.WithLabelValues(name).Set(stateToMetric(cb.State()))
	registry[name] = cb
	return cb
}

// Execute runs fn through the named breaker and records failures.
func Execute(name string, fn func() (interface{}, error)) (interface{}, error) {
	cb := Get(name)
	result, err := cb.Execute(fn)
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, fmt.Errorf("%w: %s", ErrCircuitOpen, name)
		}
		breakerFailures.WithLabelValues(name).Inc()
		return nil, err
	}
	breakerState.WithLabelValues(name).Set(stateToMetric(cb.State()))
	return result, nil
}

// State returns the current breaker state for metrics/tests.
func State(name string) gobreaker.State {
	return Get(name).State()
}

// ForceOpen trips a breaker for testing (demo only).
func ForceOpen(name string) {
	cb := Get(name)
	for i := 0; i < 4; i++ {
		_, _ = cb.Execute(func() (interface{}, error) {
			breakerFailures.WithLabelValues(name).Inc()
			return nil, errors.New("forced failure")
		})
	}
	breakerState.WithLabelValues(name).Set(stateToMetric(cb.State()))
}
