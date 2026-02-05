package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPack_TotalItems(t *testing.T) {
	tests := []struct {
		name     string
		pack     Pack
		expected int
	}{
		{
			name:     "single pack",
			pack:     Pack{Size: 500, Quantity: 1},
			expected: 500,
		},
		{
			name:     "multiple packs",
			pack:     Pack{Size: 250, Quantity: 4},
			expected: 1000,
		},
		{
			name:     "zero quantity",
			pack:     Pack{Size: 500, Quantity: 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pack.TotalItems())
		})
	}
}

func TestEmpty(t *testing.T) {
	result := Empty(100)

	assert.Equal(t, 100, result.OrderedItems)
	assert.Equal(t, 0, result.TotalItems)
	assert.Empty(t, result.Packs)
}

func TestPackResult_JSON(t *testing.T) {
	result := PackResult{
		OrderedItems: 251,
		TotalItems:   500,
		Packs: []Pack{
			{Size: 500, Quantity: 1},
		},
	}

	// Verify structure
	assert.Equal(t, 251, result.OrderedItems)
	assert.Equal(t, 500, result.TotalItems)
	assert.Len(t, result.Packs, 1)
	assert.Equal(t, 500, result.Packs[0].Size)
	assert.Equal(t, 1, result.Packs[0].Quantity)
}
