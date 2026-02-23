package ratelimiter

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewTokenBucket tests that the constructor initializes correctly
func TestNewTokenBucket(t *testing.T) {
	// Create a bucket that allows 10 operations per second with max capacity of 10
	tb := NewTokenBucket(10, time.Second, 10)

	// Verify it's not nil
	if tb == nil {
		t.Fatal("NewTokenBucket returned nil")
	}

	// Verify initial state (bucket should start full)
	if !tb.Allow() {
		t.Error("Expected first Allow() to succeed on a fresh bucket")
	}
}

// TestAllow_Success tests that Allow returns true when tokens are available
func TestAllow_Success(t *testing.T) {
	// Create a bucket with capacity for 5 tokens
	tb := NewTokenBucket(10, time.Second, 5)

	// Should be able to consume all 5 tokens
	for i := range 5 {
		if !tb.Allow() {
			t.Errorf("Allow() failed on request %d, expected to succeed", i+1)
		}
	}
}

// TestAllow_BucketEmpty tests that Allow returns false when bucket is empty
func TestAllow_BucketEmpty(t *testing.T) {
	// Create a bucket with only 2 tokens
	tb := NewTokenBucket(10, time.Second, 2)

	// Consume both tokens
	tb.Allow()
	tb.Allow()

	// Third attempt should fail (no tokens left)
	if tb.Allow() {
		t.Error("Allow() succeeded when bucket should be empty")
	}
}

// TestAllow_Refill tests that tokens refill over time
func TestAllow_Refill(t *testing.T) {
	// Create a bucket: 10 tokens per second, max 10
	tb := NewTokenBucket(10, time.Second, 10)

	// Exhaust all tokens
	for range 10 {
		tb.Allow()
	}

	// Verify bucket is empty
	if tb.Allow() {
		t.Fatal("Bucket should be empty")
	}

	// Wait for 200ms (should refill ~2 tokens at 10/second)
	time.Sleep(200 * time.Millisecond)

	// Should have at least 1 token now
	if !tb.Allow() {
		t.Error("Expected bucket to have refilled after waiting")
	}
}

// TestWait_Success tests that Wait blocks and then succeeds
func TestWait_Success(t *testing.T) {
	// Create a bucket with 1 token, refills at 10 tokens/second
	tb := NewTokenBucket(10, time.Second, 1)

	// Consume the initial token
	tb.Allow()

	// Wait should block briefly then succeed
	ctx := context.Background()
	start := time.Now()

	err := tb.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait() returned error: %v", err)
	}

	// Should have waited approximately 100ms (1 token at 10/sec = 0.1 sec)
	if elapsed < 50*time.Millisecond {
		t.Errorf("Wait() returned too quickly: %v", elapsed)
	}
}

// TestWait_ContextCancellation tests that Wait respects context cancellation
func TestWait_ContextCancellation(t *testing.T) {
	// Create a bucket with very slow refill (1 token per 10 seconds)
	tb := NewTokenBucket(1, 10*time.Second, 1)

	// Exhaust the token
	tb.Allow()

	// Create a context that cancels after 100ms
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Wait should return context error
	err := tb.Wait(ctx)

	if err == nil {
		t.Error("Expected Wait() to return error when context is cancelled")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded error, got: %v", err)
	}
}

// TestConcurrentAccess tests that multiple goroutines can safely use the bucket
func TestConcurrentAccess(t *testing.T) {
	// Create a bucket with 100 tokens
	tb := NewTokenBucket(100, time.Second, 100)

	var successCount int64 // Use int64 for atomic operations
	done := make(chan bool)

	// Launch 10 goroutines that each try to get 10 tokens
	for range 10 {
		go func() {
			for range 10 {
				if tb.Allow() {
					atomic.AddInt64(&successCount, 1) // Thread-safe increment
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}

	// All 100 attempts should succeed (we had 100 tokens)
	finalCount := atomic.LoadInt64(&successCount) // Thread-safe read
	if finalCount != 100 {
		t.Errorf("Expected 100 successes, got %d", finalCount)
	}
}

// TestMultipleInstances_IndependentState tests that multiple token buckets maintain separate state
func TestMultipleInstances_IndependentState(t *testing.T) {
	// Create 5 buckets with different configurations
	bucket1 := NewTokenBucket(10, time.Second, 5)   // 5 tokens
	bucket2 := NewTokenBucket(20, time.Second, 10)  // 10 tokens
	bucket3 := NewTokenBucket(5, time.Second, 3)    // 3 tokens
	bucket4 := NewTokenBucket(100, time.Second, 50) // 50 tokens
	bucket5 := NewTokenBucket(1, time.Second, 1)    // 1 token

	// Exhaust bucket1 completely
	for range 5 {
		bucket1.Allow()
	}

	// Exhaust bucket3 partially (use 2 out of 3)
	bucket3.Allow()
	bucket3.Allow()

	// Verify bucket1 is empty
	if bucket1.Allow() {
		t.Error("bucket1 should be empty after exhausting all tokens")
	}

	// Verify other buckets are NOT affected
	if !bucket2.Allow() {
		t.Error("bucket2 should still have tokens (independent of bucket1)")
	}
	if !bucket3.Allow() {
		t.Error("bucket3 should still have 1 token left (independent of bucket1)")
	}
	if !bucket4.Allow() {
		t.Error("bucket4 should still have tokens (independent of bucket1)")
	}
	if !bucket5.Allow() {
		t.Error("bucket5 should still have its token (independent of bucket1)")
	}

	// Verify bucket3 is now empty after using its last token
	if bucket3.Allow() {
		t.Error("bucket3 should now be empty")
	}

	// But bucket1 and bucket3 being empty shouldn't affect bucket2
	if !bucket2.Allow() {
		t.Error("bucket2 should still have tokens")
	}
}

// TestMultipleInstances_IndependentRefill tests that buckets refill at their own rates
func TestMultipleInstances_IndependentRefill(t *testing.T) {
	// Fast bucket: refills 10 tokens/second
	fastBucket := NewTokenBucket(10, time.Second, 1)

	// Slow bucket: refills 2 tokens/second
	slowBucket := NewTokenBucket(2, time.Second, 1)

	// Exhaust both buckets
	fastBucket.Allow()
	slowBucket.Allow()

	// Wait 150ms (fast should refill ~1.5 tokens, slow should refill ~0.3 tokens)
	time.Sleep(150 * time.Millisecond)

	// Fast bucket should have refilled enough for another request
	if !fastBucket.Allow() {
		t.Error("Fast bucket should have refilled by now")
	}

	// Slow bucket should NOT have refilled enough yet
	if slowBucket.Allow() {
		t.Error("Slow bucket should not have refilled enough yet")
	}

	// Wait another 400ms (total 550ms, slow bucket should have ~1.1 tokens now)
	time.Sleep(400 * time.Millisecond)

	// Now slow bucket should have refilled
	if !slowBucket.Allow() {
		t.Error("Slow bucket should have refilled by now")
	}
}
