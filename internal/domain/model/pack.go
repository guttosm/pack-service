// Package model defines the core domain entities for the pack service.
package model

// Pack represents a single pack configuration in an order fulfillment.
//
// @Description Pack size and quantity used in the order
// @Example {"size": 500, "quantity": 1}
type Pack struct {
	// Size is the pack size in items
	Size int `json:"size" example:"500"`
	// Quantity is the number of packs of this size
	Quantity int `json:"quantity" example:"1"`
}

// TotalItems returns the total number of items in this pack (size * quantity).
func (p Pack) TotalItems() int {
	return p.Size * p.Quantity
}

// PackResult represents the complete result of a pack calculation.
// It implements JSON serialization for direct use in HTTP responses.
//
// @Description Pack calculation result containing ordered items, total items shipped, and pack breakdown
// @Example {"ordered_items": 251, "total_items": 500, "packs": [{"size": 500, "quantity": 1}]}
type PackResult struct {
	// OrderedItems is the number of items the customer ordered
	OrderedItems int `json:"ordered_items" example:"251"`
	// TotalItems is the total number of items that will be shipped
	TotalItems int `json:"total_items" example:"500"`
	// Packs is the list of packs used to fulfill the order
	Packs []Pack `json:"packs"`
}

// Empty returns an empty PackResult for the given order amount.
func Empty(orderedItems int) PackResult {
	return PackResult{
		OrderedItems: orderedItems,
		TotalItems:   0,
		Packs:        []Pack{},
	}
}
