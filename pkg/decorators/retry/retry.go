// Package retry provides functionality for retrying operations with configurable backoff.
//
// This package offers a flexible approach to implementing retry mechanisms with various
// backoff strategies. It supports context cancellation, custom error handling,
// and different return signatures (error-only or value+error).
//
// Key features:
// - Configurable maximum number of attempts
// - Support for any backoff strategy implementing the Backoff interface
// - Context-aware retries that respect cancellation
// - Classification of recoverable vs. unrecoverable errors
// - Support for operations that return values
// - Callbacks for monitoring retry attempts
//
// Basic usage with error-only function:
//
//	// Create a backoff strategy
//	backoff := backoff.Default()
//
//	// Configure retry behavior
//	config := retry.DefaultConfig(backoff)
//	config.MaxAttempts = 5
//
//	// Execute with retries
//	err := retry.Do(ctx, config, func() error {
//		return someOperation()
//	})
//
// For functions that return a value and an error:
//
//	result, err := retry.DoWithValue(ctx, config, func() (MyType, error) {
//		return someOperation()
//	})
//
// To mark errors as unrecoverable (preventing further retries):
//
//	func someOperation() error {
//		// ...
//		if permanentFailure {
//			return retry.NewUnrecoverableError(fmt.Errorf("permanent failure"))
//		}
//		// ...
//	}
//
// The package integrates with the context package to support cancellation and timeouts:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	// Each retry attempt will respect the context deadline
//	err := retry.Do(ctx, config, func() error {
//		// This operation can take time but will be cancelled if the context expires
//		return timeConsumingOperation(ctx)
//	})
//
// For advanced control, you can customize the retry behavior:
//
//	config := retry.RetryConfig{
//		MaxAttempts: 10,
//		Backoff:     myCustomBackoff,
//		IsRecoverable: func(err error) bool {
//			// Custom logic to determine if an error should be retried
//			return !retry.IsUnrecoverableError(err) && !isRateLimitError(err)
//		},
//		OnRetry: func(attempt uint, err error, delay time.Duration) {
//			// Log or measure each retry attempt
//			logger.Infof("Retry %d after error: %v (waiting %v)", attempt, err, delay)
//		},
//	}
package retry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Config holds configuration for retry operations
type Config struct {
	// MaxAttempts is the maximum number of attempts before giving up
	MaxAttempts uint

	// Backoff is the backoff strategy to use
	Backoff Backoff

	// IsRecoverable is a function that determines if an error should be retried
	// If not provided, all errors except context.Canceled and unrecoverable errors will be retried
	IsRecoverable func(error) bool

	// OnRetry is an optional callback that will be called before each retry
	// The callback receives the current attempt number (starting from 1), error from the previous attempt,
	// and the delay before the next attempt
	OnRetry func(attempt uint, err error, delay time.Duration)
}

// Default returns a RetryConfig with sensible defaults
func Default(backoff Backoff) Config {
	return Config{
		MaxAttempts:   3,
		Backoff:       backoff,
		IsRecoverable: defaultRecoverable(),
	}
}

// Do executes a function with retries based on the provided config
// This is for functions that return only an error
func Do(ctx context.Context, config Config, op func() error) error {
	// Validate configuration
	if config.Backoff == nil {
		return fmt.Errorf("backoff strategy is required")
	}

	if config.MaxAttempts == 0 {
		config.MaxAttempts = 1 // At least one attempt
	}

	if config.IsRecoverable == nil {
		config.IsRecoverable = defaultRecoverable()
	}

	var lastErr error
	attempt := uint(0)
	delay := config.Backoff.MinDelay()

	for attempt < config.MaxAttempts {
		// Check context before the attempt
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute the operation
		err := op()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if context is canceled
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return err
		}

		// Check if error is recoverable
		if !config.IsRecoverable(err) {
			return err
		}

		// Increment attempt counter
		attempt++

		// Last attempt, don't delay
		if attempt >= config.MaxAttempts {
			break
		}

		// Call the OnRetry callback if provided
		if config.OnRetry != nil {
			config.OnRetry(attempt, err, delay)
		}

		// Calculate next delay and wait
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay = config.Backoff.Delay(delay)
		}
	}

	return fmt.Errorf("%w: %v", ErrAllAttemptsFailed, lastErr)
}

// DoWithValue executes a function with retries based on the provided config
// This is for functions that return a value and an error
func DoWithValue[T any](ctx context.Context, config Config, op func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	// Validate configuration
	if config.Backoff == nil {
		return zero, fmt.Errorf("backoff strategy is required")
	}

	if config.MaxAttempts == 0 {
		config.MaxAttempts = 1 // At least one attempt
	}

	if config.IsRecoverable == nil {
		config.IsRecoverable = defaultRecoverable()
	}

	attempt := uint(0)
	delay := config.Backoff.MinDelay()

	for attempt < config.MaxAttempts {
		// Check context before the attempt
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}

		// Execute the operation
		result, err := op()
		if err == nil {
			return result, nil // Success
		}

		lastErr = err

		// Check if context is canceled
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return zero, err
		}

		// Check if error is recoverable
		if !config.IsRecoverable(err) {
			return zero, err
		}

		// Increment attempt counter
		attempt++

		// Last attempt, don't delay
		if attempt >= config.MaxAttempts {
			break
		}

		// Call the OnRetry callback if provided
		if config.OnRetry != nil {
			config.OnRetry(attempt, err, delay)
		}

		// Calculate next delay and wait
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
			delay = config.Backoff.Delay(delay)
		}
	}

	return zero, fmt.Errorf("%w: %v", ErrAllAttemptsFailed, lastErr)
}

func defaultRecoverable() func(err error) bool {
	return func(err error) bool {
		return err != nil &&
			!errors.Is(err, context.Canceled) &&
			!IsUnrecoverableError(err)
	}
}
