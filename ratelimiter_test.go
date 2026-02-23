package ratelimiter

import "testing"

// Test to verify that TokenBucket implements RateLimiter interface
func TestRateLimiterInterface(t *testing.T) {
	var _ RateLimiter = (*TokenBucket)(nil)
}
