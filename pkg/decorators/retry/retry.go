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
	// Validate and prepare configuration
	if err := validateConfig(&config); err != nil {
		return err
	}

	var lastErr error

	// Run the retry loop
	err := doRetry(ctx, config, func(attempt uint) (bool, error) {
		err := op()
		if err == nil {
			return true, nil // Success
		}

		lastErr = err
		return false, err
	})

	// check if all attempts failed
	if err != nil {
		if errors.Is(err, ErrAllAttemptsFailed) {
			return fmt.Errorf("%w: %w", ErrAllAttemptsFailed, lastErr)
		}

		return err
	}

	return nil
}

// DoWithValue executes a function with retries based on the provided config
// This is for functions that return a value and an error
func DoWithValue[T any](ctx context.Context, config Config, op func() (T, error)) (T, error) {
	var zero T
	var result T
	var lastErr error

	// Validate and prepare configuration
	if err := validateConfig(&config); err != nil {
		return zero, err
	}

	// Run the retry loop
	err := doRetry(ctx, config, func(attempt uint) (bool, error) {
		var err error
		result, err = op()
		if err == nil {
			return true, nil // Success
		}

		lastErr = err
		return false, err
	})

	// If we have an actual error from the retry mechanism, return it
	if err != nil {
		if errors.Is(err, ErrAllAttemptsFailed) {
			return zero, fmt.Errorf("%w: %v", ErrAllAttemptsFailed, lastErr)
		}

		return zero, err
	}

	// Otherwise return the successful result
	return result, nil
}

// validateConfig checks and initializes the retry configuration
func validateConfig(config *Config) error {
	if config.Backoff == nil {
		return fmt.Errorf("backoff strategy is required")
	}

	if config.MaxAttempts == 0 {
		config.MaxAttempts = 1 // At least one attempt
	}

	if config.IsRecoverable == nil {
		config.IsRecoverable = defaultRecoverable()
	}

	return nil
}

// doRetry implements the core retry logic
// The operation function returns a boolean indicating success and an error
func doRetry(ctx context.Context, config Config, operation func(attempt uint) (bool, error)) error {
	attempt := uint(0)
	delay := config.Backoff.MinDelay()

	for attempt < config.MaxAttempts {
		// Check context before the attempt
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute the operation
		success, err := operation(attempt)
		if success {
			return nil // Operation succeeded
		}

		// Check if context is canceled or deadline exceeded
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) ||
			ctx.Err() != nil {
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

	// We've exhausted all attempts
	return ErrAllAttemptsFailed
}

func defaultRecoverable() func(err error) bool {
	return func(err error) bool {
		return err != nil &&
			!errors.Is(err, context.Canceled) &&
			!errors.Is(err, context.DeadlineExceeded) &&
			!IsUnrecoverableError(err)
	}
}
