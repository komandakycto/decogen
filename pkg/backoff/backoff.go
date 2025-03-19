// Package backoff provides implementations of backoff algorithms for retry mechanisms.
//
// Backoff algorithms are used to determine the delay between retry attempts,
// typically increasing the delay after each failed attempt. This package implements
// exponential backoff with jitter, which is useful for distributed systems to
// prevent the "thundering herd" problem where many clients retry simultaneously.
//
// Exponential backoff increases the retry interval exponentially, providing
// progressively longer waits between retries. Jitter adds a random component
// to the delay, which helps distribute retry attempts over time.
//
// The BackOff implementation in this package is thread-safe and can be safely
// used by multiple goroutines concurrently.
//
// Example usage:
//
//	// Create a backoff with custom parameters
//	b := backoff.New(
//		100*time.Millisecond, // Min delay
//		10*time.Second,       // Max delay
//		2.0,                  // Multiplication factor
//		0.1,                  // Jitter factor
//	)
//
//	// Or use the default configuration
//	b := backoff.Default()
//
//	// Use in a retry loop
//	delay := b.MinDelay()
//	for attempt := 0; attempt < maxAttempts; attempt++ {
//		err := operation()
//		if err == nil {
//			break // Operation successful
//		}
//
//		// Wait before next attempt
//		time.Sleep(delay)
//		delay = b.Delay(delay)
//	}
package backoff

import (
	"math/rand"
	"sync"
	"time"
)

// BackOff implements exponential backoff with jitter
type BackOff struct {
	minDelay time.Duration
	maxDelay time.Duration
	factor   float64
	jitter   float64
	rnd      *rand.Rand
	mu       sync.Mutex // protects rnd
}

// New creates a new instance of BackOff
func New(minDelay, maxDelay time.Duration, factor, jitter float64) *BackOff {
	// Create a local random source with a unique seed
	source := rand.NewSource(time.Now().UnixNano())
	return &BackOff{
		minDelay: minDelay,
		maxDelay: maxDelay,
		factor:   factor,
		jitter:   jitter,
		rnd:      rand.New(source),
	}
}

// Default creates a BackOff with sensible defaults
func Default() *BackOff {
	return New(
		100*time.Millisecond, // Min delay
		10*time.Second,       // Max delay
		2.0,                  // Multiplication factor
		0.1,                  // Jitter factor (as a percentage of delay)
	)
}

// MinDelay returns the minimum configured delay
func (b *BackOff) MinDelay() time.Duration {
	return b.minDelay
}

// MaxDelay returns the maximum configured delay
func (b *BackOff) MaxDelay() time.Duration {
	return b.maxDelay
}

// Factor returns the multiplication factor for backoff
func (b *BackOff) Factor() float64 {
	return b.factor
}

// Jitter returns the jitter factor
func (b *BackOff) Jitter() float64 {
	return b.jitter
}

// Delay calculates the next backoff delay using exponential backoff with jitter
func (b *BackOff) Delay(previous time.Duration) time.Duration {
	// Ensure we're starting with at least minDelay
	if previous < b.minDelay {
		previous = b.minDelay
	}

	// Calculate exponential backoff
	delay := time.Duration(float64(previous) * b.factor)

	// Cap at maxDelay before adding jitter
	if delay > b.maxDelay {
		delay = b.maxDelay
	}

	// Add jitter (random variation to avoid thundering herd)
	b.mu.Lock()
	// Generate a random value in range [-jitter/2, jitter/2]
	jitterFactor := (b.rnd.Float64() - 0.5) * b.jitter
	b.mu.Unlock()

	// Apply jitter as a percentage of current delay
	jitterAmount := time.Duration(float64(delay) * jitterFactor)
	delay += jitterAmount

	// Ensure we don't go below minDelay or above maxDelay after jitter
	if delay < b.minDelay {
		delay = b.minDelay
	} else if delay > b.maxDelay {
		delay = b.maxDelay
	}

	return delay
}
