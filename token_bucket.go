package ratelimiter

import (
	"context"
	"sync"
	"time"
)

type TokenBucket struct {
	// Our token bucket struct that keeps track of request/token capacity
	mtx         sync.Mutex // our lock for thread safety
	rate        float64    // tokens added per second
	max_tokens  float64    // maximum token capacity for our bucket; using float64 instead of int just to prevent the need of casting in the math later
	tokens      float64    // current count of available tokens; using float64 since our rate will refill the tokens fractionally
	lastUpdated time.Time  // last time tokens were updated
}

func NewTokenBucket(maxOps int, per time.Duration, maxBucketSize int) *TokenBucket {
	// This constructor allows us to pass in any rate we want, and then standardizes it
	// to the rate per second

	rate := float64(maxOps) / per.Seconds()

	return &TokenBucket{
		rate:        rate,
		max_tokens:  float64(maxBucketSize),
		tokens:      float64(maxBucketSize),
		lastUpdated: time.Now(),
	}
}

// Implements Allow RateLimiter method to determine whether we allow or deny incoming event/request
// Returns true if we have available tokens, and false if no tokens are available (bucket is empty)
// NON-BLOCKING! Returns immediately
func (tb *TokenBucket) Allow() bool {
	// First, we establish our lock + unlock mechanism for concurrency safety
	tb.mtx.Lock()
	defer tb.mtx.Unlock() // ensures we don't accidentally forget to unlock somewhere

	// Next, refill bucket to ensure we're up to date on the current token state
	tb.refillBucket()

	// Check if we have enough tokens in our bucket for our event/request in the bucket -- we need at least 1 full token
	if tb.tokens >= 1 {
		tb.tokens-- // use up 1 token
		return true
	}
	return false
}

// Implements Wait RateLimiter method which blocks an event/request until we have enough capacity
// It returns an error if the context is canceled
// BLOCKING!! Blocks current goroutine
func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		// Try to get a token
		tb.mtx.Lock()
		tb.refillBucket()

		if tb.tokens >= 1 {
			tb.tokens--
			tb.mtx.Unlock()
			return nil // Success! Token acquired
		}

		// Otherwise, no token available - calculate how long to wait
		tokensNeeded := 1.0 - tb.tokens
		waitDuration := time.Duration(tokensNeeded / tb.rate * float64(time.Second))
		tb.mtx.Unlock() // unlock here so other goroutines can access rate limiter if needed

		// Wait for that duration OR context cancellation
		select {
		case <-time.After(waitDuration):
			// Time passed, loop again to try acquiring token
			continue
		case <-ctx.Done():
			// Context cancelled - return error
			return ctx.Err()
		}
	}
}

// Internal helper function to add token capacity to bucket based on our refill rate until max capacity is hit
func (tb *TokenBucket) refillBucket() {
	// Figure out elapsed time since last event/request
	now := time.Now()
	elapsed := now.Sub(tb.lastUpdated).Seconds()

	// Add tokens based on elapsed time using our rate
	tb.tokens += elapsed * tb.rate

	// Cap at max token/bucket limit
	if tb.tokens > tb.max_tokens {
		tb.tokens = tb.max_tokens
	}

	tb.lastUpdated = now
}
