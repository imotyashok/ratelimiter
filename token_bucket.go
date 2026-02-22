package ratelimiter

import (
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
