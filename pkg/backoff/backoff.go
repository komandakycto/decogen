package backoff

import (
	"math/rand"
	"time"
)

// BackOff implements exponential backoff with jitter
type BackOff struct {
	minDelay time.Duration
	maxDelay time.Duration
	factor   float64
	jitter   float64
}

// NewBackOff creates a new instance of BackOff
func NewBackOff(minDelay, maxDelay time.Duration, factor, jitter float64) *BackOff {
	return &BackOff{
		minDelay: minDelay,
		maxDelay: maxDelay,
		factor:   factor,
		jitter:   jitter,
	}
}

// DefaultBackOff creates a BackOff with sensible defaults
func DefaultBackOff() *BackOff {
	return NewBackOff(
		100*time.Millisecond, // Min delay
		10*time.Second,       // Max delay
		2.0,                  // Multiplication factor
		0.1,                  // Jitter factor (as a percentage of second)
	)
}

// MinDelay returns the minimum configured delay
func (b *BackOff) MinDelay() time.Duration {
	return b.minDelay
}

// Delay calculates the next backoff delay using exponential backoff with jitter
func (b *BackOff) Delay(previous time.Duration) time.Duration {
	// Ensure we're starting with at least minDelay
	if previous < b.minDelay {
		previous = b.minDelay
	}

	// Calculate exponential backoff
	delay := time.Duration(float64(previous) * b.factor)

	// Cap at maxDelay
	if delay > b.maxDelay {
		delay = b.maxDelay
	}

	// Add jitter (random variation to avoid thundering herd)
	jitterRange := time.Duration(b.jitter * float64(time.Second))
	jitterAmount := time.Duration(rand.Float64()*float64(jitterRange) - float64(jitterRange/2))
	delay += jitterAmount

	// Ensure we don't go below minDelay due to negative jitter
	if delay < b.minDelay {
		delay = b.minDelay
	}

	return delay
}
