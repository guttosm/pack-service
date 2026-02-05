//go:build !integration

package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := New(DefaultConfig())
	err := cb.Execute(context.Background(), func() error {
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
		Name:             "test",
	})

	testErr := errors.New("test error")

	// First failure
	err := cb.Execute(context.Background(), func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateClosed, cb.State())

	// Second failure - should open circuit
	err = cb.Execute(context.Background(), func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateOpen, cb.State())

	err = cb.Execute(context.Background(), func() error {
		return nil // This won't be called
	})
	assert.Equal(t, ErrCircuitOpen, err)
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		Name:             "test",
	})

	// Open the circuit
	_ = cb.Execute(context.Background(), func() error {
		return errors.New("error")
	})
	_ = cb.Execute(context.Background(), func() error {
		return errors.New("error")
	})
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// First success in half-open
	err := cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())

	// Second success - should close circuit
	err = cb.Execute(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_HalfOpen_Failure(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		Name:             "test",
	})

	// Open the circuit
	_ = cb.Execute(context.Background(), func() error {
		return errors.New("error")
	})
	_ = cb.Execute(context.Background(), func() error {
		return errors.New("error")
	})
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Failure in half-open should immediately reopen
	err := cb.Execute(context.Background(), func() error {
		return errors.New("error")
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreaker_GetStats(t *testing.T) {
	cb := New(DefaultConfig())

	stats := cb.GetStats()
	assert.Equal(t, "closed", stats.State)
	assert.True(t, stats.IsHealthy)
	assert.Equal(t, 0, stats.FailureCount)

	// Cause a failure
	_ = cb.Execute(context.Background(), func() error {
		return errors.New("error")
	})

	stats = cb.GetStats()
	assert.Equal(t, 1, stats.FailureCount)
}

func TestCircuitBreaker_IsOpen(t *testing.T) {
	cb := New(Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
		Name:             "test",
	})

	assert.False(t, cb.IsOpen())

	_ = cb.Execute(context.Background(), func() error {
		return errors.New("error")
	})

	assert.True(t, cb.IsOpen())
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, 5, config.FailureThreshold)
	assert.Equal(t, 2, config.SuccessThreshold)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, "circuit-breaker", config.Name)
}
