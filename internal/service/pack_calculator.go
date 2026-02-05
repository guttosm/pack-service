package service

import (
	"sort"
	"sync"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/service/cache"
)

var (
	// DefaultPackSizes defines the standard pack sizes available for orders.
	DefaultPackSizes = []int{5000, 2000, 1000, 500, 250}
)

// dpState holds the dynamic programming arrays for reuse via sync.Pool.
// This significantly reduces allocations for high-volume traffic.
type dpState struct {
	dp     []int
	parent []int
}

// dpPool provides reusable DP state objects to reduce GC pressure.
// Pre-allocates for common case (10K items).
var dpPool = sync.Pool{
	New: func() interface{} {
		return &dpState{
			dp:     make([]int, 0, 10000),
			parent: make([]int, 0, 10000),
		}
	},
}

// getDPState retrieves a dpState from the pool and ensures it has sufficient capacity.
func getDPState(size int) *dpState {
	state, _ := dpPool.Get().(*dpState)
	if state == nil {
		state = &dpState{
			dp:     make([]int, 0, 10000),
			parent: make([]int, 0, 10000),
		}
	}

	// Resize if needed
	if cap(state.dp) < size {
		state.dp = make([]int, size)
		state.parent = make([]int, size)
	} else {
		state.dp = state.dp[:size]
		state.parent = state.parent[:size]
	}

	// Reset values
	for i := range state.dp {
		state.dp[i] = -1
		state.parent[i] = -1
	}
	state.dp[0] = 0

	return state
}

// putDPState returns a dpState to the pool for reuse.
func putDPState(state *dpState) {
	// Clear references to help GC if slices are very large
	if cap(state.dp) > 100000 {
		state.dp = make([]int, 0, 10000)
		state.parent = make([]int, 0, 10000)
	}
	dpPool.Put(state)
}

// PackCalculator defines the interface for pack calculation operations.
type PackCalculator interface {
	Calculate(itemsOrdered int) model.PackResult
	CalculateWithPackSizes(itemsOrdered int, packSizes []int) model.PackResult
	// InvalidateCache clears the calculation cache (useful when pack sizes change)
	InvalidateCache()
}

// Option configures a PackCalculatorService.
type Option func(*PackCalculatorService)

// PackCalculatorService implements PackCalculator using an optimized algorithm.
// It uses dynamic programming to find the optimal combination of packs
// that minimizes total items while using the fewest number of packs.
type PackCalculatorService struct {
	packSizes    []int
	smallestPack int
	cache        cache.Cache
}

// NewPackCalculatorService creates a new PackCalculatorService with the given options.
func NewPackCalculatorService(opts ...Option) *PackCalculatorService {
	s := &PackCalculatorService{
		packSizes: make([]int, len(DefaultPackSizes)),
	}
	copy(s.packSizes, DefaultPackSizes)

	for _, opt := range opts {
		opt(s)
	}

	s.smallestPack = s.packSizes[len(s.packSizes)-1]
	return s
}

// WithPackSizes sets custom pack sizes for the calculator.
func WithPackSizes(sizes []int) Option {
	return func(s *PackCalculatorService) {
		if len(sizes) > 0 {
			s.packSizes = make([]int, len(sizes))
			copy(s.packSizes, sizes)
			sort.Sort(sort.Reverse(sort.IntSlice(s.packSizes)))
		}
	}
}

// WithCache enables result caching with the specified capacity and TTL.
func WithCache(capacity int, ttl time.Duration) Option {
	return func(s *PackCalculatorService) {
		if capacity > 0 {
			s.cache = newTTLCache(capacity, ttl)
		}
	}
}

// WithCacheInterface allows injecting a custom cache implementation.
func WithCacheInterface(c cache.Cache) Option {
	return func(s *PackCalculatorService) {
		s.cache = c
	}
}

// Calculate determines the optimal packs needed for the given order.
func (s *PackCalculatorService) Calculate(itemsOrdered int) model.PackResult {
	if itemsOrdered <= 0 {
		return model.Empty(itemsOrdered)
	}

	if s.cache != nil {
		if result, ok := s.cache.Get(itemsOrdered); ok {
			return result
		}
	}

	result := s.calculateCore(itemsOrdered, s.packSizes, s.smallestPack)

	if s.cache != nil {
		s.cache.Set(itemsOrdered, result)
	}

	return result
}

// CalculateWithPackSizes calculates packs using custom pack sizes provided in the request.
func (s *PackCalculatorService) CalculateWithPackSizes(itemsOrdered int, packSizes []int) model.PackResult {
	if itemsOrdered <= 0 {
		return model.Empty(itemsOrdered)
	}

	if len(packSizes) == 0 {
		return s.Calculate(itemsOrdered)
	}

	tempSizes := make([]int, len(packSizes))
	copy(tempSizes, packSizes)
	sort.Sort(sort.Reverse(sort.IntSlice(tempSizes)))

	smallestPack := tempSizes[len(tempSizes)-1]
	return s.calculateCore(itemsOrdered, tempSizes, smallestPack)
}

// calculateCore is the unified DP algorithm implementation.
// It uses sync.Pool for slice reuse to minimize allocations.
func (s *PackCalculatorService) calculateCore(target int, packSizes []int, smallestPack int) model.PackResult {
	if len(packSizes) == 0 {
		return model.Empty(target)
	}

	// Handle small orders efficiently without DP
	if target <= smallestPack {
		return s.smallOrderWithSizes(target, packSizes, smallestPack)
	}

	maxItems := target + smallestPack - 1

	// Get pooled DP state
	state := getDPState(maxItems + 1)
	defer putDPState(state)

	dp := state.dp
	parent := state.parent

	// Dynamic programming
	for i := 0; i <= maxItems; i++ {
		if dp[i] == -1 {
			continue
		}
		for _, packSize := range packSizes {
			next := i + packSize
			if next > maxItems {
				continue
			}
			newPacks := dp[i] + 1
			if dp[next] == -1 || newPacks < dp[next] {
				dp[next] = newPacks
				parent[next] = packSize
			}
		}
		// Early exit optimization
		if i >= target && dp[i] != -1 {
			hasBetter := false
			for j := i + 1; j <= maxItems && j < i+smallestPack; j++ {
				if dp[j] != -1 && dp[j] < dp[i] {
					hasBetter = true
					break
				}
			}
			if !hasBetter {
				break
			}
		}
	}

	// Find minimum items >= target
	minItems := -1
	for items := target; items <= maxItems && items < target+smallestPack; items++ {
		if dp[items] != -1 {
			minItems = items
			break
		}
	}

	if minItems == -1 {
		return model.Empty(target)
	}

	return s.buildResultWithSizes(target, minItems, parent, packSizes)
}

// smallOrderWithSizes handles very small orders efficiently without DP.
func (s *PackCalculatorService) smallOrderWithSizes(target int, packSizes []int, smallestPack int) model.PackResult {
	// Find smallest pack that fits (pack sizes are sorted descending)
	for i := len(packSizes) - 1; i >= 0; i-- {
		size := packSizes[i]
		if size >= target {
			return model.PackResult{
				OrderedItems: target,
				TotalItems:   size,
				Packs:        []model.Pack{{Size: size, Quantity: 1}},
			}
		}
	}
	// If no pack fits, use smallest pack
	return model.PackResult{
		OrderedItems: target,
		TotalItems:   smallestPack,
		Packs:        []model.Pack{{Size: smallestPack, Quantity: 1}},
	}
}

// buildResultWithSizes constructs the PackResult from DP solution.
// Uses array-based counting instead of map for better performance.
func (s *PackCalculatorService) buildResultWithSizes(target, minItems int, parent []int, packSizes []int) model.PackResult {
	// Use array indexed by position instead of map for small sets
	packCounts := make([]int, len(packSizes))
	sizeToIndex := make(map[int]int, len(packSizes))
	for i, size := range packSizes {
		sizeToIndex[size] = i
	}

	// Backtrack to count packs used
	for curr := minItems; curr > 0; {
		packSize := parent[curr]
		if idx, ok := sizeToIndex[packSize]; ok {
			packCounts[idx]++
		}
		curr -= packSize
	}

	// Build result slice
	packs := make([]model.Pack, 0, len(packSizes))
	for i, count := range packCounts {
		if count > 0 {
			packs = append(packs, model.Pack{Size: packSizes[i], Quantity: count})
		}
	}

	return model.PackResult{
		OrderedItems: target,
		TotalItems:   minItems,
		Packs:        packs,
	}
}

// InvalidateCache clears the calculation cache.
func (s *PackCalculatorService) InvalidateCache() {
	if s.cache != nil {
		if cacheWithClear, ok := s.cache.(interface{ Clear() }); ok {
			cacheWithClear.Clear()
		}
	}
}
