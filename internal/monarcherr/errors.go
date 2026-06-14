package monarcherr

import (
	"fmt"
)

type Code string

const (
	AuthRequired         Code = "AUTH_REQUIRED"
	AuthSessionExpired   Code = "AUTH_SESSION_EXPIRED"
	AuthMFARequired      Code = "AUTH_MFA_REQUIRED"
	AuthMFAInvalid       Code = "AUTH_MFA_INVALID"
	NetworkUnreachable   Code = "NETWORK_UNREACHABLE"
	NetworkTimeout       Code = "NETWORK_TIMEOUT"
	APIError             Code = "API_ERROR"
	APISchemaChanged     Code = "API_SCHEMA_CHANGED"
	FeatureUnavailable   Code = "FEATURE_UNAVAILABLE"
	ValidationFailed     Code = "VALIDATION_FAILED"
	ReadOnlyViolation    Code = "READ_ONLY_VIOLATION"
	ConfirmationRequired Code = "CONFIRMATION_REQUIRED"
	ResourceNotFound     Code = "RESOURCE_NOT_FOUND"
	InternalError        Code = "INTERNAL_ERROR"
	InvalidArguments     Code = "INVALID_ARGUMENTS"
)

type Category string

const (
	CatAuth       Category = "auth"
	CatNetwork    Category = "network"
	CatAPI        Category = "api"
	CatValidation Category = "validation"
	CatInternal   Category = "internal"
)

type Error struct {
	Code      Code     `json:"code"`
	Message   string   `json:"message"`
	Category  Category `json:"category"`
	Retryable bool     `json:"retryable"`
	Err       error    `json:"-"`
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code Code, message string, category Category, retryable bool, err error) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Category:  category,
		Retryable: retryable,
		Err:       err,
	}
}
