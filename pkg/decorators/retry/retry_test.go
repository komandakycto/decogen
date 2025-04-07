package retry_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/komandakycto/decogen/pkg/decorators/retry"
)

// MockBackoff implements the retry.Backoff interface for testing
type MockBackoff struct {
	mock.Mock
}

func (m *MockBackoff) MinDelay() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockBackoff) Delay(previous time.Duration) time.Duration {
	args := m.Called(previous)
	return args.Get(0).(time.Duration)
}

// TestDo tests the basic retry functionality
func TestDo(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)

		attempts := 0
		err := retry.Do(context.Background(), retry.Config{
			MaxAttempts: 3,
			Backoff:     mockB,
		}, func() error {
			attempts++
			return nil // Success
		})

		require.NoError(t, err)
		require.Equal(t, 1, attempts, "Operation should be called exactly once")
		mockB.AssertExpectations(t)
	})

	t.Run("success after retries", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)
		mockB.On("Delay", mock.Anything).Return(20 * time.Millisecond).Times(2)

		attempts := 0
		err := retry.Do(context.Background(), retry.Config{
			MaxAttempts: 5,
			Backoff:     mockB,
		}, func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil // Success on third attempt
		})

		require.NoError(t, err)
		require.Equal(t, 3, attempts, "Operation should be called exactly 3 times")
		mockB.AssertExpectations(t)
	})

	t.Run("failure after all attempts", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)
		mockB.On("Delay", mock.Anything).Return(20 * time.Millisecond).Times(2)

		attempts := 0
		err := retry.Do(context.Background(), retry.Config{
			MaxAttempts: 3,
			Backoff:     mockB,
		}, func() error {
			attempts++
			return errors.New("persistent error")
		})

		require.Error(t, err)
		require.ErrorIs(t, err, retry.ErrAllAttemptsFailed)
		require.Contains(t, err.Error(), "persistent error")
		require.Equal(t, 3, attempts, "Operation should be called exactly 3 times")
		mockB.AssertExpectations(t)
	})
}

// TestDoWithValue tests DoWithValue function
func TestDoWithValue(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)

		attempts := 0
		result, err := retry.DoWithValue(context.Background(), retry.Config{
			MaxAttempts: 3,
			Backoff:     mockB,
		}, func() (string, error) {
			attempts++
			return "success", nil
		})

		require.NoError(t, err)
		require.Equal(t, "success", result)
		require.Equal(t, 1, attempts, "Operation should be called exactly once")
		mockB.AssertExpectations(t)
	})

	t.Run("success after retries", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)
		mockB.On("Delay", mock.Anything).Return(20 * time.Millisecond).Times(1)

		attempts := 0
		result, err := retry.DoWithValue(context.Background(), retry.Config{
			MaxAttempts: 3,
			Backoff:     mockB,
		}, func() (string, error) {
			attempts++
			if attempts < 2 {
				return "", errors.New("temporary error")
			}
			return "eventual success", nil
		})

		require.NoError(t, err)
		require.Equal(t, "eventual success", result)
		require.Equal(t, 2, attempts, "Operation should be called exactly 2 times")
		mockB.AssertExpectations(t)
	})

	t.Run("failure after all attempts", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)
		mockB.On("Delay", mock.Anything).Return(20 * time.Millisecond).Times(2)

		attempts := 0
		result, err := retry.DoWithValue(context.Background(), retry.Config{
			MaxAttempts: 3,
			Backoff:     mockB,
		}, func() (string, error) {
			attempts++
			return "", errors.New("persistent error")
		})

		require.Error(t, err)
		require.ErrorIs(t, err, retry.ErrAllAttemptsFailed)
		require.Contains(t, err.Error(), "persistent error")
		require.Equal(t, "", result, "Failed operation should return zero value")
		require.Equal(t, 3, attempts, "Operation should be called exactly 3 times")
		mockB.AssertExpectations(t)
	})
}

// TestErrorHandling tests error handling functionality
func TestErrorHandling(t *testing.T) {
	t.Run("unrecoverable error stops retries", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)

		attempts := 0
		err := retry.Do(context.Background(), retry.Config{
			MaxAttempts: 3,
			Backoff:     mockB,
		}, func() error {
			attempts++
			return retry.NewUnrecoverableError(errors.New("critical error"))
		})

		require.Error(t, err)
		require.True(t, retry.IsUnrecoverableError(err))
		require.Contains(t, err.Error(), "critical error")
		require.Equal(t, 1, attempts, "Operation should be called exactly once")
		mockB.AssertExpectations(t)
	})

	t.Run("context cancellation stops retries", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)
		mockB.On("Delay", mock.Anything).Return(20 * time.Millisecond).Maybe()

		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0

		// Use a channel to coordinate test timing
		operationStarted := make(chan struct{})

		// Run in goroutine since we'll cancel the context
		errCh := make(chan error)
		go func() {
			errCh <- retry.Do(ctx, retry.Config{
				MaxAttempts: 5,
				Backoff:     mockB,
				OnRetry: func(attempt uint, err error, delay time.Duration) {
					if attempt == 1 {
						close(operationStarted)
					}
				},
			}, func() error {
				attempts++
				return errors.New("temporary error")
			})
		}()

		// Wait for the first attempt to complete
		select {
		case <-operationStarted:
			// Good, operation has started
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Operation didn't start within expected time")
		}

		// Cancel the context
		cancel()

		// Get the result
		err := <-errCh

		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
		require.GreaterOrEqual(t, attempts, 1, "Operation should be called at least once")
		require.LessOrEqual(t, attempts, 2, "Operation should not be called more than twice")
	})

	t.Run("custom recoverable function", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)
		// Add expected call to Delay even though we don't expect the retry to happen
		// This handles a potential edge case in the implementation
		mockB.On("Delay", mock.Anything).Return(20 * time.Millisecond).Maybe()

		attempts := 0
		err := retry.Do(context.Background(), retry.Config{
			MaxAttempts: 5,
			Backoff:     mockB,
			IsRecoverable: func(err error) bool {
				// Only retry errors containing "retry"
				return err != nil && !strings.Contains(err.Error(), "retry")
			},
		}, func() error {
			attempts++
			return errors.New("do not retry this")
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "do not retry this")
		require.Equal(t, 1, attempts, "Operation should be called exactly once")
		mockB.AssertExpectations(t)
	})
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	t.Run("missing backoff", func(t *testing.T) {
		err := retry.Do(context.Background(), retry.Config{
			MaxAttempts: 3,
			Backoff:     nil, // Missing backoff
		}, func() error {
			return nil
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "backoff strategy is required")
	})

	t.Run("zero max attempts", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)

		attempts := 0
		err := retry.Do(context.Background(), retry.Config{
			MaxAttempts: 0, // Should be adjusted to 1
			Backoff:     mockB,
		}, func() error {
			attempts++
			return nil
		})

		require.NoError(t, err)
		require.Equal(t, 1, attempts, "Operation should be called exactly once")
		mockB.AssertExpectations(t)
	})

	t.Run("default config", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)

		config := retry.Default(mockB)

		require.Equal(t, uint(3), config.MaxAttempts)
		require.Equal(t, mockB, config.Backoff)
		require.NotNil(t, config.IsRecoverable)
	})
}

// TestWithTimeout tests the WithTimeout function
func TestWithTimeout(t *testing.T) {
	t.Run("operation completes within timeout", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)

		result, err := retry.WithTimeout(
			context.Background(),
			retry.Config{
				MaxAttempts: 3,
				Backoff:     mockB,
			},
			100*time.Millisecond,
			func(ctx context.Context) (string, error) {
				// Operation completes quickly
				return "success", nil
			},
		)

		require.NoError(t, err)
		require.Equal(t, "success", result)
		mockB.AssertExpectations(t)
	})

	t.Run("operation exceeds timeout", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)

		result, err := retry.WithTimeout(
			context.Background(),
			retry.Config{
				MaxAttempts: 3,
				Backoff:     mockB,
			},
			50*time.Millisecond,
			func(ctx context.Context) (string, error) {
				// This operation takes longer than the timeout
				select {
				case <-time.After(200 * time.Millisecond):
					return "too late", nil
				case <-ctx.Done():
					return "", ctx.Err()
				}
			},
		)

		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, "", result)
		mockB.AssertExpectations(t)
	})

	t.Run("invalid timeout", func(t *testing.T) {
		mockB := new(MockBackoff)

		_, err := retry.WithTimeout(
			context.Background(),
			retry.Config{
				MaxAttempts: 3,
				Backoff:     mockB,
			},
			0, // Invalid timeout
			func(ctx context.Context) (string, error) {
				return "success", nil
			},
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "timeout must be greater than zero")
	})
}

// TestHelperFunctions tests the helper function
func TestHelperFunctions(t *testing.T) {
	t.Run("WithLogging adds logging", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)
		mockB.On("Delay", mock.Anything).Return(20 * time.Millisecond).Times(2)

		// Create a logger that writes to a buffer
		var buf bytes.Buffer
		logger := log.New(&buf, "", 0)

		config := retry.Config{
			MaxAttempts: 3,
			Backoff:     mockB,
		}

		// Add logging
		configWithLogging := retry.WithLogging(config, logger)

		// Should log retries
		err := retry.Do(context.Background(), configWithLogging, func() error {
			return errors.New("test error")
		})

		require.Error(t, err)
		require.ErrorIs(t, err, retry.ErrAllAttemptsFailed)

		// Check that something was logged
		logOutput := buf.String()
		require.NotEmpty(t, logOutput, "Should have logged retry attempts")
		require.Contains(t, logOutput, "Retry attempt")
		require.Contains(t, logOutput, "test error")
	})

	t.Run("IsTemporary detects temporary errors", func(t *testing.T) {
		// Regular error
		regularErr := errors.New("regular error")
		require.False(t, retry.IsTemporary(regularErr))

		// Temporary error
		tempErr := &temporaryTestError{isTemp: true}
		require.True(t, retry.IsTemporary(tempErr))

		// Non-temporary error that implements the interface
		nonTempErr := &temporaryTestError{isTemp: false}
		require.False(t, retry.IsTemporary(nonTempErr))
	})

	t.Run("WithTemporaryErrorHandling retries temporary errors", func(t *testing.T) {
		mockB := new(MockBackoff)
		mockB.On("MinDelay").Return(10 * time.Millisecond)
		mockB.On("Delay", mock.Anything).Return(20 * time.Millisecond).Maybe()

		config := retry.Config{
			MaxAttempts: 3,
			Backoff:     mockB,
			// By default, we would NOT retry the error
			IsRecoverable: func(err error) bool {
				return false
			},
		}

		// Use WithTemporaryErrorHandling
		configWithTempHandling := retry.WithTemporaryErrorHandling(config)

		attempts := 0
		err := retry.Do(context.Background(), configWithTempHandling, func() error {
			attempts++
			if attempts == 1 {
				// Return a temporary error on first attempt
				return &temporaryTestError{isTemp: true}
			}
			return nil
		})

		require.NoError(t, err)
		require.Equal(t, 2, attempts, "Should have retried the temporary error")
	})
}

// TestErrorUnwrapping tests error unwrapping
func TestErrorUnwrapping(t *testing.T) {
	t.Run("unrecoverable error unwrapping", func(t *testing.T) {
		originalErr := errors.New("original error")
		unrecoverableErr := retry.NewUnrecoverableError(originalErr)

		// Check if we can unwrap to the original error
		unwrappedErr := errors.Unwrap(unrecoverableErr)
		require.Equal(t, originalErr, unwrappedErr)

		// Test with wrapped error
		wrappedErr := fmt.Errorf("wrapped: %w", unrecoverableErr)
		require.True(t, retry.IsUnrecoverableError(wrappedErr))
	})
}

// Helper type for testing

// temporaryTestError implements retry.IsTemporaryError
type temporaryTestError struct {
	isTemp bool
}

func (e *temporaryTestError) Error() string {
	return "temporary test error"
}

func (e *temporaryTestError) Temporary() bool {
	return e.isTemp
}
