package backoff_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/komandakycto/decogen/pkg/backoff"
)

func TestNew(t *testing.T) {
	// Test with specific parameters
	minDelay := 50 * time.Millisecond
	maxDelay := 5 * time.Second
	factor := 1.5
	jitter := 0.2

	b := backoff.New(minDelay, maxDelay, factor, jitter)

	assert.Equal(t, minDelay, b.MinDelay(), "minDelay should match the input value")
	assert.Equal(t, maxDelay, b.MaxDelay(), "maxDelay should match the input value")
	assert.Equal(t, factor, b.Factor(), "factor should match the input value")
	assert.Equal(t, jitter, b.Jitter(), "jitter should match the input value")
}

func TestDefault(t *testing.T) {
	b := backoff.Default()

	assert.Equal(t, 100*time.Millisecond, b.MinDelay(), "default minDelay should be 100ms")
	assert.Equal(t, 10*time.Second, b.MaxDelay(), "default maxDelay should be 10s")
	assert.Equal(t, 2.0, b.Factor(), "default factor should be 2.0")
	assert.Equal(t, 0.1, b.Jitter(), "default jitter should be 0.1")
}

func TestMinDelayAndMaxDelay(t *testing.T) {
	minDelay := 200 * time.Millisecond
	maxDelay := 30 * time.Second
	b := backoff.New(minDelay, maxDelay, 2.0, 0.1)

	assert.Equal(t, minDelay, b.MinDelay(), "MinDelay() should return the configured minimum delay")
	assert.Equal(t, maxDelay, b.MaxDelay(), "MaxDelay() should return the configured maximum delay")
}

func TestDelay_RespectsMinDelay(t *testing.T) {
	minDelay := 100 * time.Millisecond
	maxDelay := 10 * time.Second
	b := backoff.New(minDelay, maxDelay, 2.0, 0.0) // Zero jitter for deterministic testing

	// Test with previous value less than minDelay
	delay := b.Delay(50 * time.Millisecond)
	assert.GreaterOrEqual(t, delay, minDelay, "Delay should be at least minDelay when previous is less than minDelay")
}

func TestDelay_RespectsMaxDelay(t *testing.T) {
	minDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second
	b := backoff.New(minDelay, maxDelay, 2.0, 0.0) // Zero jitter for deterministic testing

	// Test with value that would exceed maxDelay after factoring
	delay := b.Delay(600 * time.Millisecond)
	assert.LessOrEqual(t, delay, maxDelay, "Delay should not exceed maxDelay")
}

func TestDelay_ExponentialGrowth(t *testing.T) {
	minDelay := 100 * time.Millisecond
	maxDelay := 10 * time.Second
	factor := 2.0
	b := backoff.New(minDelay, maxDelay, factor, 0.0) // Zero jitter for deterministic testing

	delay := minDelay
	for i := 0; i < 5; i++ {
		newDelay := b.Delay(delay)
		if float64(delay) < float64(maxDelay)/factor {
			expectedDelay := time.Duration(float64(delay) * factor)
			assert.Equal(t, expectedDelay, newDelay,
				"Delay should increase by factor of %f when not hitting max", factor)
		}
		delay = newDelay
	}
}

func TestDelay_WithJitter(t *testing.T) {
	minDelay := 100 * time.Millisecond
	maxDelay := 10 * time.Second
	factor := 2.0
	jitter := 0.5 // Large jitter for visible effect
	b := backoff.New(minDelay, maxDelay, factor, jitter)

	// Collect multiple delay samples with the same previous value
	sampleSize := 100
	samples := make([]time.Duration, sampleSize)

	previousDelay := 200 * time.Millisecond
	expectedBaseDelay := time.Duration(float64(previousDelay) * factor)

	// Generate samples
	for i := 0; i < sampleSize; i++ {
		samples[i] = b.Delay(previousDelay)
	}

	// Verify samples are not all identical (jitter is working)
	uniqueValues := make(map[time.Duration]bool)
	for _, sample := range samples {
		uniqueValues[sample] = true
	}
	assert.Greater(t, len(uniqueValues), 1, "Jitter should produce varying delays")

	// Calculate mean and verify it's close to the expected base delay
	var sum time.Duration
	for _, sample := range samples {
		sum += sample
	}
	mean := sum / time.Duration(sampleSize)

	// Allow for some statistical variation, but the mean should be close to expected
	allowedDeviation := float64(expectedBaseDelay) * 0.1 // 10% tolerance
	assert.InDelta(t, float64(expectedBaseDelay), float64(mean), allowedDeviation,
		"Mean of jittered delays should be close to expected base delay")

	// Verify all samples respect min/max bounds
	for _, sample := range samples {
		assert.GreaterOrEqual(t, sample, minDelay, "All delays should be >= minDelay")
		assert.LessOrEqual(t, sample, maxDelay, "All delays should be <= maxDelay")
	}
}

func TestDelay_ConcurrentSafety(t *testing.T) {
	minDelay := 100 * time.Millisecond
	maxDelay := 10 * time.Second
	b := backoff.New(minDelay, maxDelay, 2.0, 0.5)

	// Run multiple goroutines that call Delay concurrently
	concurrentCalls := 100
	done := make(chan bool)

	for i := 0; i < concurrentCalls; i++ {
		go func(iter int) {
			// Call Delay multiple times from each goroutine
			previousDelay := time.Duration(100+iter) * time.Millisecond
			for j := 0; j < 10; j++ {
				delay := b.Delay(previousDelay)
				assert.GreaterOrEqual(t, delay, minDelay, "Concurrent delay should respect minDelay")
				assert.LessOrEqual(t, delay, maxDelay, "Concurrent delay should respect maxDelay")
				previousDelay = delay
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrentCalls; i++ {
		<-done
	}
}

func TestDelay_EdgeCases(t *testing.T) {
	t.Run("zero min delay", func(t *testing.T) {
		b := backoff.New(0, time.Second, 2.0, 0.0)
		delay := b.Delay(0)
		assert.Equal(t, time.Duration(0), delay, "With zero minDelay and zero previous, should return zero")
	})

	t.Run("negative previous delay", func(t *testing.T) {
		minDelay := 100 * time.Millisecond
		b := backoff.New(minDelay, time.Second, 2.0, 0.0)
		delay := b.Delay(-100 * time.Millisecond)
		assert.GreaterOrEqual(t, delay, minDelay, "With negative previous delay, should use minDelay")
	})

	t.Run("zero factor", func(t *testing.T) {
		minDelay := 100 * time.Millisecond
		b := backoff.New(minDelay, time.Second, 0.0, 0.0)
		delay := b.Delay(minDelay)
		assert.Equal(t, minDelay, delay, "With zero factor, should return previous or minDelay")
	})

	t.Run("equal min and max delay", func(t *testing.T) {
		sameDelay := 500 * time.Millisecond
		b := backoff.New(sameDelay, sameDelay, 2.0, 0.0)
		delay := b.Delay(sameDelay)
		assert.Equal(t, sameDelay, delay, "With equal min and max, should always return that value")
	})

	t.Run("extremely large factor", func(t *testing.T) {
		minDelay := 100 * time.Millisecond
		maxDelay := time.Second
		b := backoff.New(minDelay, maxDelay, 1000.0, 0.0)
		delay := b.Delay(minDelay)
		assert.Equal(t, maxDelay, delay, "With very large factor, should cap at maxDelay")
	})
}

func TestDelay_StatisticalDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping statistical tests in short mode")
	}

	minDelay := 100 * time.Millisecond
	maxDelay := time.Second
	factor := 2.0
	jitter := 0.2
	b := backoff.New(minDelay, maxDelay, factor, jitter)

	previousDelay := 200 * time.Millisecond
	expectedBase := time.Duration(float64(previousDelay) * factor)

	// Generate large sample for statistical analysis
	sampleSize := 10000
	samples := make([]time.Duration, sampleSize)

	for i := 0; i < sampleSize; i++ {
		samples[i] = b.Delay(previousDelay)
	}

	// Calculate mean
	var sum time.Duration
	for _, s := range samples {
		sum += s
	}
	mean := float64(sum) / float64(sampleSize)

	// Verify mean is close to expected base
	assert.InDelta(t, float64(expectedBase), mean, float64(expectedBase)*0.05,
		"Mean should be close to expected base delay")

	// Calculate variance and standard deviation
	var sumSquareDiff float64
	for _, s := range samples {
		diff := float64(s) - mean
		sumSquareDiff += diff * diff
	}
	variance := sumSquareDiff / float64(sampleSize)
	stdDev := math.Sqrt(variance)

	// Expected standard deviation with uniform jitter of size jitter*delay
	expectedStdDev := float64(expectedBase) * jitter / math.Sqrt(12)

	// Allow some tolerance in the standard deviation comparison
	assert.InDelta(t, expectedStdDev, stdDev, expectedStdDev*0.5,
		"Standard deviation should be close to expected value for uniform distribution")
}
