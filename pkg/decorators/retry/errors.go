package retry

import (
	"errors"
	"fmt"
)

// Common errors returned by retry operations
var (
	// ErrAllAttemptsFailed is returned when all retry attempts have been exhausted
	ErrAllAttemptsFailed = errors.New("all retry attempts failed")
)

// UnrecoverableError wraps an error to indicate that it should not be retried
type UnrecoverableError struct {
	cause error
}

// NewUnrecoverableError wraps an error to indicate it should not be retried
func NewUnrecoverableError(err error) error {
	if err == nil {
		return nil
	}
	return &UnrecoverableError{cause: err}
}

// Error implements the error interface
func (e *UnrecoverableError) Error() string {
	if e.cause == nil {
		return "unrecoverable error"
	}
	return fmt.Sprintf("unrecoverable: %v", e.cause)
}

// Unwrap returns the wrapped error
func (e *UnrecoverableError) Unwrap() error {
	return e.cause
}

// IsUnrecoverableError checks if an error or any error in its chain is marked as unrecoverable
func IsUnrecoverableError(err error) bool {
	var unrecoverableErr *UnrecoverableError
	return errors.As(err, &unrecoverableErr)
}
