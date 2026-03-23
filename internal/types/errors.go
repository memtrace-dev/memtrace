package types

import "errors"

var (
	ErrNotInitialized = errors.New("memtrace is not initialized in this directory — run 'memtrace init' first")
	ErrMemoryNotFound = errors.New("memory not found")
	ErrValidation     = errors.New("validation error")
)

// ValidationError wraps ErrValidation with field-level detail.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}
