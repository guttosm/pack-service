// Package circuitbreaker provides circuit breaker pattern implementation for resilience.
package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrInvalidState indicates an invalid circuit breaker state transition.
	ErrInvalidState = errors.New("invalid circuit breaker state")
)

// State represents the state of the circuit breaker.
type State int

const (
	// StateClosed means the circuit is closed and requests pass through normally.
	StateClosed State = iota
	// StateOpen means the circuit is open and requests are rejected immediately.
	StateOpen
	// StateHalfOpen means the circuit is half-open, allowing a test request.
	StateHalfOpen
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration.
type Config struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit.
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes needed to close the circuit.
	SuccessThreshold int
	// Timeout is the duration to wait before attempting to half-open the circuit.
	Timeout time.Duration
	// Name is the name of the circuit breaker (for logging).
	Name string
}

// DefaultConfig returns a default circuit breaker configuration.
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		Name:             "circuit-breaker",
	}
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config          Config
	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	mu              sync.RWMutex
}

// New creates a new circuit breaker with the given configuration.
func New(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// Execute executes a function with circuit breaker protection.
// Returns ErrCircuitOpen if the circuit is open.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if we should transition from open to half-open
	cb.mu.Lock()
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			log.Info().
				Str("circuit_breaker", cb.config.Name).
				Msg("Circuit breaker transitioning to half-open")
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}
	cb.mu.Unlock()

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// onFailure handles a failure.
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = StateOpen
			log.Warn().
				Str("circuit_breaker", cb.config.Name).
				Int("failure_count", cb.failureCount).
				Msg("Circuit breaker opened due to failures")
		}
	case StateHalfOpen:
		// Any failure in half-open state immediately opens the circuit
		cb.state = StateOpen
		cb.failureCount = cb.config.FailureThreshold
		log.Warn().
			Str("circuit_breaker", cb.config.Name).
			Msg("Circuit breaker reopened after half-open failure")
	}
}

// onSuccess handles a success.
func (cb *CircuitBreaker) onSuccess() {
	cb.failureCount = 0

	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.successCount = 0
			log.Info().
				Str("circuit_breaker", cb.config.Name).
				Msg("Circuit breaker closed after successful recovery")
		}
	case StateClosed:
		// Reset success count in closed state
		cb.successCount = 0
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// IsOpen returns true if the circuit breaker is open.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == StateOpen
}

// Stats returns circuit breaker statistics.
type Stats struct {
	State         string
	FailureCount  int
	SuccessCount  int
	LastFailure   time.Time
	IsHealthy     bool
}

// GetStats returns current circuit breaker statistics.
func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Stats{
		State:         cb.state.String(),
		FailureCount:  cb.failureCount,
		SuccessCount:  cb.successCount,
		LastFailure:   cb.lastFailureTime,
		IsHealthy:     cb.state == StateClosed,
	}
}
