package ratelimiter

import "context"

// RateLimiter interface; all algorithms must implement this.
type RateLimiter interface {

	// Non-blocking check that returns true/false if event/request can pass or not
	Allow() bool

	// Blocks until allowed or context cancelled
	Wait(ctx context.Context) error
}
