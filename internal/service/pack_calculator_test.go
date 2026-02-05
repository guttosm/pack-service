package service

import (
	"sync"
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/stretchr/testify/assert"
)

// TestNewPackCalculatorService tests the constructor and options.
func TestNewPackCalculatorService(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		validate func(*testing.T, *PackCalculatorService)
	}{
		{
			name:    "uses default pack sizes when no options",
			options: nil,
			validate: func(t *testing.T, svc *PackCalculatorService) {
				assert.Equal(t, DefaultPackSizes, svc.packSizes)
			},
		},
		{
			name:    "uses custom pack sizes with option",
			options: []Option{WithPackSizes([]int{100, 500, 250})},
			validate: func(t *testing.T, svc *PackCalculatorService) {
				assert.Equal(t, []int{500, 250, 100}, svc.packSizes)
			},
		},
		{
			name:    "enables cache with option",
			options: []Option{WithCache(100, 5*time.Minute)},
			validate: func(t *testing.T, svc *PackCalculatorService) {
				assert.NotNil(t, svc.cache)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewPackCalculatorService(tt.options...)
			if tt.validate != nil {
				tt.validate(t, svc)
			}
		})
	}
}

// TestPackCalculatorService_Calculate tests the core calculation logic.
func TestPackCalculatorService_Calculate(t *testing.T) {
	svc := NewPackCalculatorService()

	tests := []struct {
		name          string
		itemsOrdered  int
		expectedTotal int
		expectedPacks []model.Pack
	}{
		{
			name:          "1 item returns 1x250",
			itemsOrdered:  1,
			expectedTotal: 250,
			expectedPacks: []model.Pack{{Size: 250, Quantity: 1}},
		},
		{
			name:          "250 items returns 1x250",
			itemsOrdered:  250,
			expectedTotal: 250,
			expectedPacks: []model.Pack{{Size: 250, Quantity: 1}},
		},
		{
			name:          "251 items returns 1x500",
			itemsOrdered:  251,
			expectedTotal: 500,
			expectedPacks: []model.Pack{{Size: 500, Quantity: 1}},
		},
		{
			name:          "501 items returns 1x500 + 1x250",
			itemsOrdered:  501,
			expectedTotal: 750,
			expectedPacks: []model.Pack{{Size: 500, Quantity: 1}, {Size: 250, Quantity: 1}},
		},
		{
			name:          "12001 items returns 2x5000 + 1x2000 + 1x250",
			itemsOrdered:  12001,
			expectedTotal: 12250,
			expectedPacks: []model.Pack{{Size: 5000, Quantity: 2}, {Size: 2000, Quantity: 1}, {Size: 250, Quantity: 1}},
		},
		{
			name:          "0 items returns empty",
			itemsOrdered:  0,
			expectedTotal: 0,
			expectedPacks: []model.Pack{},
		},
		{
			name:          "negative items returns empty",
			itemsOrdered:  -10,
			expectedTotal: 0,
			expectedPacks: []model.Pack{},
		},
		{
			name:          "exact 500 returns 1x500",
			itemsOrdered:  500,
			expectedTotal: 500,
			expectedPacks: []model.Pack{{Size: 500, Quantity: 1}},
		},
		{
			name:          "exact 5000 returns 1x5000",
			itemsOrdered:  5000,
			expectedTotal: 5000,
			expectedPacks: []model.Pack{{Size: 5000, Quantity: 1}},
		},
		{
			name:          "750 returns 1x500 + 1x250",
			itemsOrdered:  750,
			expectedTotal: 750,
			expectedPacks: []model.Pack{{Size: 500, Quantity: 1}, {Size: 250, Quantity: 1}},
		},
		{
			name:          "10000 returns 2x5000",
			itemsOrdered:  10000,
			expectedTotal: 10000,
			expectedPacks: []model.Pack{{Size: 5000, Quantity: 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.Calculate(tt.itemsOrdered)

			assert.Equal(t, tt.itemsOrdered, result.OrderedItems)
			assert.Equal(t, tt.expectedTotal, result.TotalItems)
			assert.Equal(t, tt.expectedPacks, result.Packs)

			// Verify sum matches
			var sum int
			for _, p := range result.Packs {
				sum += p.Size * p.Quantity
			}
			assert.Equal(t, result.TotalItems, sum)
		})
	}
}

// TestPackCalculatorService_EdgeCases tests boundary conditions.
func TestPackCalculatorService_EdgeCases(t *testing.T) {
	svc := NewPackCalculatorService()

	tests := []struct {
		name         string
		itemsOrdered int
		validate     func(*testing.T, model.PackResult)
	}{
		{
			name:         "exactly smallest pack",
			itemsOrdered: 250,
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 250, result.OrderedItems)
				assert.Equal(t, 250, result.TotalItems)
				assert.Len(t, result.Packs, 1)
			},
		},
		{
			name:         "one less than pack size",
			itemsOrdered: 249,
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 249, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 249)
			},
		},
		{
			name:         "very large number",
			itemsOrdered: 1000000,
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 1000000, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 1000000)
				assert.NotEmpty(t, result.Packs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.Calculate(tt.itemsOrdered)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestPackCalculatorService_CustomPackSizes tests calculation with custom pack sizes.
func TestPackCalculatorService_CustomPackSizes(t *testing.T) {
	tests := []struct {
		name         string
		packSizes    []int
		itemsOrdered int
		validate     func(*testing.T, model.PackResult)
	}{
		{
			name:         "custom pack sizes calculation",
			packSizes:    []int{100, 300, 500},
			itemsOrdered: 150,
			validate: func(t *testing.T, result model.PackResult) {
				assert.GreaterOrEqual(t, result.TotalItems, 150)
				assert.LessOrEqual(t, result.TotalItems, 200) // 2x100 = 200
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewPackCalculatorService(WithPackSizes(tt.packSizes))
			result := svc.Calculate(tt.itemsOrdered)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestPackCalculatorService_WithCacheInterface tests cache integration with mock.
func TestPackCalculatorService_WithCacheInterface(t *testing.T) {
	tests := []struct {
		name         string
		itemsOrdered int
		setupMock    func(*mocks.MockCache)
		validate     func(*testing.T, model.PackResult)
	}{
		{
			name:         "cache miss then cache set",
			itemsOrdered: 251,
			setupMock: func(mockCache *mocks.MockCache) {
				mockCache.EXPECT().Get(251).Return(model.PackResult{}, false).Once()
				mockCache.EXPECT().Set(251, model.PackResult{
					OrderedItems: 251,
					TotalItems:   500,
					Packs:        []model.Pack{{Size: 500, Quantity: 1}},
				}).Once()
			},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 500, result.TotalItems)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := mocks.NewMockCache(t)
			tt.setupMock(mockCache)

			svc := NewPackCalculatorService(WithCacheInterface(mockCache))
			result := svc.Calculate(tt.itemsOrdered)

			if tt.validate != nil {
				tt.validate(t, result)
			}
			mockCache.AssertExpectations(t)
		})
	}
}

// TestPackCalculatorService_Cache tests basic cache functionality.
func TestPackCalculatorService_Cache(t *testing.T) {
	tests := []struct {
		name         string
		itemsOrdered int
		validate     func(*testing.T, *PackCalculatorService, int)
	}{
		{
			name:         "cache hit on second call",
			itemsOrdered: 251,
			validate: func(t *testing.T, svc *PackCalculatorService, itemsOrdered int) {
				// First call - cache miss
				result1 := svc.Calculate(itemsOrdered)
				assert.Equal(t, 500, result1.TotalItems)

				// Second call - cache hit (same result)
				result2 := svc.Calculate(itemsOrdered)
				assert.Equal(t, result1, result2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewPackCalculatorService(WithCache(10, 5*time.Minute))
			if tt.validate != nil {
				tt.validate(t, svc, tt.itemsOrdered)
			}
		})
	}
}

// TestPackCalculatorService_CacheTTL tests cache expiration.
func TestPackCalculatorService_CacheTTL(t *testing.T) {
	tests := []struct {
		name         string
		itemsOrdered int
		cacheTTL     time.Duration
		sleepTime    time.Duration
		validate     func(*testing.T, *PackCalculatorService, int)
	}{
		{
			name:         "cache expires after TTL",
			itemsOrdered: 251,
			cacheTTL:     50 * time.Millisecond,
			sleepTime:    100 * time.Millisecond,
			validate: func(t *testing.T, svc *PackCalculatorService, itemsOrdered int) {
				// First call
				result1 := svc.Calculate(itemsOrdered)
				assert.Equal(t, 500, result1.TotalItems)

				// Wait for TTL to expire
				time.Sleep(100 * time.Millisecond)

				// Cache should have expired, but result should still be correct
				result2 := svc.Calculate(itemsOrdered)
				assert.Equal(t, result1, result2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewPackCalculatorService(WithCache(10, tt.cacheTTL))
			if tt.validate != nil {
				tt.validate(t, svc, tt.itemsOrdered)
			}
		})
	}
}

// TestPackCalculatorService_CacheConcurrency tests cache under concurrent access.
func TestPackCalculatorService_CacheConcurrency(t *testing.T) {
	tests := []struct {
		name          string
		goroutines    int
		calculateFunc func(int) int
		validate      func(*testing.T, *PackCalculatorService, int)
	}{
		{
			name:       "concurrent cache access",
			goroutines: 100,
			calculateFunc: func(n int) int {
				return n * 100
			},
			validate: func(t *testing.T, svc *PackCalculatorService, goroutines int) {
				var wg sync.WaitGroup
				for i := 0; i < goroutines; i++ {
					wg.Add(1)
					go func(n int) {
						defer wg.Done()
						itemsOrdered := (n + 1) * 100
						result := svc.Calculate(itemsOrdered)
						assert.GreaterOrEqual(t, result.TotalItems, itemsOrdered)
					}(i)
				}
				wg.Wait()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewPackCalculatorService(WithCache(100, 5*time.Minute))
			if tt.validate != nil {
				tt.validate(t, svc, tt.goroutines)
			}
		})
	}
}

// Benchmarks

func BenchmarkCalculate_Small(b *testing.B) {
	svc := NewPackCalculatorService()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Calculate(251)
	}
}

func BenchmarkCalculate_Medium(b *testing.B) {
	svc := NewPackCalculatorService()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Calculate(12001)
	}
}

func BenchmarkCalculate_Large(b *testing.B) {
	svc := NewPackCalculatorService()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Calculate(100000)
	}
}

func BenchmarkCalculate_WithCache(b *testing.B) {
	svc := NewPackCalculatorService(WithCache(1000, 5*time.Minute))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Calculate(12001)
	}
}

func BenchmarkCalculate_Parallel(b *testing.B) {
	svc := NewPackCalculatorService(WithCache(1000, 5*time.Minute))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			svc.Calculate((i%100 + 1) * 100)
			i++
		}
	})
}

// TestPackCalculatorService_CalculateWithPackSizes tests calculation with custom pack sizes.
func TestPackCalculatorService_CalculateWithPackSizes(t *testing.T) {
	svc := NewPackCalculatorService()

	tests := []struct {
		name         string
		itemsOrdered int
		packSizes    []int
		validate     func(*testing.T, model.PackResult)
	}{
		{
			name:         "zero items",
			itemsOrdered: 0,
			packSizes:    []int{100, 50, 25},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 0, result.OrderedItems)
				assert.Equal(t, 0, result.TotalItems)
				assert.Empty(t, result.Packs)
			},
		},
		{
			name:         "negative items",
			itemsOrdered: -10,
			packSizes:    []int{100, 50, 25},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, -10, result.OrderedItems)
				assert.Equal(t, 0, result.TotalItems)
				assert.Empty(t, result.Packs)
			},
		},
		{
			name:         "empty pack sizes falls back to default",
			itemsOrdered: 251,
			packSizes:    []int{},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 251, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 251)
				assert.NotEmpty(t, result.Packs)
			},
		},
		{
			name:         "nil pack sizes falls back to default",
			itemsOrdered: 251,
			packSizes:    nil,
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 251, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 251)
				assert.NotEmpty(t, result.Packs)
			},
		},
		{
			name:         "custom pack sizes sorted correctly",
			itemsOrdered: 75,
			packSizes:    []int{25, 50, 100},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 75, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 75)
				assert.NotEmpty(t, result.Packs)
				total := 0
				for _, p := range result.Packs {
					total += p.Size * p.Quantity
				}
				assert.Equal(t, result.TotalItems, total)
			},
		},
		{
			name:         "custom pack sizes already sorted",
			itemsOrdered: 75,
			packSizes:    []int{100, 50, 25},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 75, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 75)
				assert.NotEmpty(t, result.Packs)
			},
		},
		{
			name:         "exact match with custom sizes",
			itemsOrdered: 50,
			packSizes:    []int{50, 25},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 50, result.OrderedItems)
				assert.Equal(t, 50, result.TotalItems)
				assert.Len(t, result.Packs, 1)
				assert.Equal(t, 50, result.Packs[0].Size)
				assert.Equal(t, 1, result.Packs[0].Quantity)
			},
		},
		{
			name:         "large order with custom sizes",
			itemsOrdered: 1000,
			packSizes:    []int{500, 200, 100},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 1000, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 1000)
				assert.NotEmpty(t, result.Packs)
				total := 0
				for _, p := range result.Packs {
					total += p.Size * p.Quantity
				}
				assert.Equal(t, result.TotalItems, total)
			},
		},
		{
			name:         "single pack size",
			itemsOrdered: 75,
			packSizes:    []int{50},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 75, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 75)
				assert.NotEmpty(t, result.Packs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.CalculateWithPackSizes(tt.itemsOrdered, tt.packSizes)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestPackCalculatorService_CalculateWithPackSizesInternal tests the CalculateWithPackSizes method with various inputs.
func TestPackCalculatorService_CalculateWithPackSizesInternal(t *testing.T) {
	svc := NewPackCalculatorService()

	tests := []struct {
		name      string
		target    int
		packSizes []int
		validate  func(*testing.T, model.PackResult)
	}{
		{
			name:      "empty pack sizes uses defaults",
			target:    100,
			packSizes: []int{},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 100, result.OrderedItems)
				// With empty pack sizes, falls back to default Calculate
				assert.GreaterOrEqual(t, result.TotalItems, 100)
			},
		},
		{
			name:      "target less than smallest pack",
			target:    10,
			packSizes: []int{100, 50, 25},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 10, result.OrderedItems)
				assert.Equal(t, 25, result.TotalItems)
				assert.Len(t, result.Packs, 1)
				assert.Equal(t, 25, result.Packs[0].Size)
			},
		},
		{
			name:      "target equals smallest pack",
			target:    25,
			packSizes: []int{100, 50, 25},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 25, result.OrderedItems)
				assert.Equal(t, 25, result.TotalItems)
				assert.Len(t, result.Packs, 1)
				assert.Equal(t, 25, result.Packs[0].Size)
			},
		},
		{
			name:      "target between pack sizes",
			target:    75,
			packSizes: []int{100, 50, 25},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 75, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 75)
				assert.LessOrEqual(t, result.TotalItems, 75+24)
				assert.NotEmpty(t, result.Packs)
			},
		},
		{
			name:      "exact match with larger pack",
			target:    100,
			packSizes: []int{100, 50, 25},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 100, result.OrderedItems)
				assert.Equal(t, 100, result.TotalItems)
				assert.Len(t, result.Packs, 1)
				assert.Equal(t, 100, result.Packs[0].Size)
			},
		},
		{
			name:      "requires multiple packs",
			target:    150,
			packSizes: []int{100, 50, 25},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 150, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 150)
				assert.LessOrEqual(t, result.TotalItems, 150+24)
				total := 0
				for _, p := range result.Packs {
					total += p.Size * p.Quantity
				}
				assert.Equal(t, result.TotalItems, total)
			},
		},
		{
			name:      "large target",
			target:    10000,
			packSizes: []int{5000, 2000, 1000, 500, 250},
			validate: func(t *testing.T, result model.PackResult) {
				assert.Equal(t, 10000, result.OrderedItems)
				assert.GreaterOrEqual(t, result.TotalItems, 10000)
				assert.LessOrEqual(t, result.TotalItems, 10000+249)
				total := 0
				for _, p := range result.Packs {
					total += p.Size * p.Quantity
				}
				assert.Equal(t, result.TotalItems, total)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.CalculateWithPackSizes(tt.target, tt.packSizes)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
