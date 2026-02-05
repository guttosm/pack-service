// Package i18n provides internationalization support for the pack service.
package i18n

// Error message translation keys.
const (
	// ErrKeyInvalidRequest indicates an invalid request.
	ErrKeyInvalidRequest = "error.invalid_request"
	// ErrKeyInvalidRequestBody indicates an invalid request body.
	ErrKeyInvalidRequestBody = "error.invalid_request_body"
	// ErrKeyInternalError indicates an internal server error.
	ErrKeyInternalError = "error.internal_error"
	// ErrKeyUnauthorized indicates missing or invalid authentication.
	ErrKeyUnauthorized = "error.unauthorized"
	// ErrKeyInvalidCredentials indicates invalid login credentials (user not registered or wrong password).
	ErrKeyInvalidCredentials = "error.invalid_credentials"
	// ErrKeyAPIKeyRequired indicates that an API key is required.
	ErrKeyAPIKeyRequired = "error.api_key_required"
	// ErrKeyInvalidAPIKey indicates an invalid API key.
	ErrKeyInvalidAPIKey = "error.invalid_api_key"
	// ErrKeyForbidden indicates insufficient permissions.
	ErrKeyForbidden = "error.forbidden"
	// ErrKeyNotFound indicates a resource was not found.
	ErrKeyNotFound = "error.not_found"
	// ErrKeyRateLimitExceeded indicates rate limit exceeded.
	ErrKeyRateLimitExceeded = "error.rate_limit_exceeded"
	// ErrKeyConflict indicates a conflict with current state.
	ErrKeyConflict = "error.conflict"
	// ErrKeyValidationItemsOrdered indicates invalid items_ordered validation.
	ErrKeyValidationItemsOrdered = "error.validation.items_ordered"
	// ErrKeyInvalidToken indicates an invalid or expired JWT token.
	ErrKeyInvalidToken = "error.invalid_token"
	// ErrKeyTokenRequired indicates that a JWT token is required.
	ErrKeyTokenRequired = "error.token_required"
	// ErrKeyTimeout indicates a request timeout.
	ErrKeyTimeout = "error.timeout"
)

// Success message translation keys.
const (
	// SuccessKeyPackCalculated indicates successful pack calculation.
	SuccessKeyPackCalculated = "success.pack_calculated"
)
