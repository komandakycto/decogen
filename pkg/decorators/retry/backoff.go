package retry

import "time"

// Backoff defines the interface for backoff strategies
type Backoff interface {
	// MinDelay returns the minimum delay duration
	MinDelay() time.Duration

	// Delay calculates the next delay based on the previous delay
	Delay(previous time.Duration) time.Duration
}
