// Package dto defines Data Transfer Objects for HTTP request and response handling.
//
// DTOs are used to decouple the HTTP layer from the domain model,
// providing validation and serialization for API communication.
package dto

// CalculatePacksRequest represents the JSON request body for the pack calculation endpoint.
//
// The ItemsOrdered field is required and must be a positive integer.
// PackSizes is optional - if not provided, uses server-configured pack sizes.
// Validation is performed using gin's binding tags.
//
// @Description Request to calculate optimal pack combination for an order
// @Example {"items_ordered": 251}
// @Example {"items_ordered": 251, "pack_sizes": [23, 31, 53]}
type CalculatePacksRequest struct {
	// ItemsOrdered is the number of items the customer wants to order.
	// Must be greater than 0.
	ItemsOrdered int `json:"items_ordered" binding:"required,gt=0" example:"251" minimum:"1"`
	// PackSizes is an optional list of pack sizes to use for calculation.
	// If not provided, uses server-configured pack sizes.
	PackSizes []int `json:"pack_sizes" example:"23,31,53"`
} // @name CalculatePacksRequest

// ValidationError represents a field validation error.
type ValidationError struct {
	Field   string
	Message string
}

var (
	// ErrInvalidItemsOrdered is returned when items_ordered is invalid.
	ErrInvalidItemsOrdered = &ValidationError{
		Field:   "items_ordered",
		Message: "must be a positive integer",
	}
)

// Validate performs custom validation on the request.
// Returns an error if validation fails, nil otherwise.
func (r *CalculatePacksRequest) Validate() error {
	if r.ItemsOrdered <= 0 {
		return ErrInvalidItemsOrdered
	}
	return nil
}

// Error returns the error message for ValidationError.
func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// UpdatePackSizesRequest represents the JSON request body for updating pack sizes.
type UpdatePackSizesRequest struct {
	// Sizes is the list of pack sizes to use.
	Sizes []int `json:"sizes" binding:"required,min=1"`
	// CreatedBy is the identifier of who created this configuration.
	CreatedBy string `json:"created_by,omitempty"`
} // @name UpdatePackSizesRequest
