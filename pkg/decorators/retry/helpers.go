package retry

import (
	"context"
	"fmt"
	"log"
	"time"
)

// WithTimeout retries an operation with a timeout for each attempt
// The provided context applies to the entire retry process
func WithTimeout[T any](ctx context.Context, config Config, timeout time.Duration, op func(context.Context) (T, error)) (T, error) {
	var zero T

	if timeout <= 0 {
		return zero, fmt.Errorf("timeout must be greater than zero")
	}

	wrappedOp := func() (T, error) {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		return op(timeoutCtx)
	}

	return DoWithValue(ctx, config, wrappedOp)
}

// WithLogging wraps a Config to add logging on each retry
func WithLogging(config Config, logger *log.Logger) Config {
	originalOnRetry := config.OnRetry

	config.OnRetry = func(attempt uint, err error, delay time.Duration) {
		if logger != nil {
			logger.Printf("Retry attempt %d after error: %v (waiting %v)", attempt, err, delay)
		}

		if originalOnRetry != nil {
			originalOnRetry(attempt, err, delay)
		}
	}

	return config
}

// IsTemporaryError is an interface for errors that can indicate if they're temporary
type IsTemporaryError interface {
	Temporary() bool
}

// IsTemporary checks if an error indicates it's temporary/transient
func IsTemporary(err error) bool {
	if temp, ok := err.(IsTemporaryError); ok {
		return temp.Temporary()
	}
	return false
}

// WithTemporaryErrorHandling enhances a retry config to handle temporary errors
// Errors that implement IsTemporaryError interface will be retried if Temporary() returns true
func WithTemporaryErrorHandling(config Config) Config {
	originalIsRecoverable := config.IsRecoverable

	config.IsRecoverable = func(err error) bool {
		// Check if it's a temporary error
		if IsTemporary(err) {
			return true
		}

		// Fall back to the original check if provided
		if originalIsRecoverable != nil {
			return originalIsRecoverable(err)
		}

		// Default behavior
		return err != nil && !IsUnrecoverableError(err)
	}

	return config
}
