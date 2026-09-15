package domain

import "fmt"

// AppError is a machine-readable application error.
type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code, message string, httpStatus int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

var (
	ErrNotFound         = NewAppError("NOT_FOUND", "Resource not found", 404, nil)
	ErrEndpointNotFound = NewAppError("ENDPOINT_NOT_FOUND", "Endpoint not found", 404, nil)
	ErrEventNotFound    = NewAppError("EVENT_NOT_FOUND", "Event not found", 404, nil)
	ErrDeliveryNotFound = NewAppError("DELIVERY_NOT_FOUND", "Delivery not found", 404, nil)
	ErrUnauthorized     = NewAppError("UNAUTHORIZED", "Unauthorized", 401, nil)
	ErrForbidden        = NewAppError("FORBIDDEN", "Forbidden", 403, nil)
	ErrInvalidSignature = NewAppError("INVALID_SIGNATURE", "Invalid webhook signature", 401, nil)
	ErrInvalidTimestamp = NewAppError("INVALID_TIMESTAMP", "Invalid or expired timestamp", 401, nil)
	ErrInvalidPayload   = NewAppError("INVALID_PAYLOAD", "Invalid request payload", 400, nil)
	ErrPayloadTooLarge  = NewAppError("PAYLOAD_TOO_LARGE", "Payload exceeds maximum size", 413, nil)
	ErrRateLimited      = NewAppError("RATE_LIMITED", "Rate limit exceeded", 429, nil)
	ErrEndpointInactive = NewAppError("ENDPOINT_INACTIVE", "Endpoint is inactive", 403, nil)
	ErrInvalidURL       = NewAppError("INVALID_URL", "Target URL is not allowed", 400, nil)
	ErrConflict         = NewAppError("CONFLICT", "Resource already exists", 409, nil)
	ErrValidation       = NewAppError("VALIDATION_ERROR", "Validation failed", 400, nil)
)
