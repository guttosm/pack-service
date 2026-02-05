//go:build integration

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestPackCalculatorService_CalculateIntegration tests integration with real cache.
// This test verifies that the cache works correctly with actual cache implementation.
func TestPackCalculatorService_CalculateIntegration(t *testing.T) {
	svc := NewPackCalculatorService(WithCache(100, 5*time.Minute))

	// Test that cache works with calculation
	result1 := svc.Calculate(251)
	result2 := svc.Calculate(251) // Should use cache

	assert.Equal(t, result1, result2)
	assert.Equal(t, 500, result1.TotalItems)
}
