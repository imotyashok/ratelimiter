## Intro, How to Test, and How to Use
Rate Limiter submission by Iryna Motyashok

This is a demo rate limiter that implements the Token Bucket algorithm, written in Go. It's thread safe, in memory, and allows two different modes of operation: a non-blocking `Allow()` and a blocking `Wait()`.  

To test, simply run:
`go test -v` 

To use in your code:
```go
  // Allow 10 requests per second with burst capacity of 20
  // NOTE: you can specify any rate (per second, minute, hour, etc). Must be non-zero!
  limiter := NewTokenBucket(10, time.Second, 20)

  // Non-blocking check
  if limiter.Allow() {
      // Process request
  } else {
      // Reject request (429 Too Many Requests)
  }

  // Blocking wait; set your context with timeout
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()

  if err := limiter.Wait(ctx); err != nil {
      // Context timeout or cancelled
  } else {
      // Proceed with request
  }
```

## Algorithm explanation
I implemented a Token Bucket algorithm for my rate limiter because I thought:
- It's a good overall multi-purpose rate limiting algorithm, almost like a jack of all trades
- It handles an influx or burst requests/events 
- It's widely adopted in production systems, from preventing against DDoS attacks to managing bandwidth for an internet service provider

It works by having a maximum token bucket capacity which gets depleted from as each new event/request comes in and a token gets consumed. If we consumed all of our tokens, it won't let the event/request through (or rather, it will let you know that the request is not allowed to come through). 

Basically, if a bucket has tokens available then we allow the request through, and if the bucket is empty, we reject the request.

We can specify a refill rate so that the bucket doesn't just stay empty forever. Instead, tokens get added back into our bucket until we reach our maximum bucket capacity so that we can continue to consume tokens/allow new requests through. 


## Design rationale
Key decisions made:
1. RateLimiter serves as the interface, and algorithms (like TokenBucket) can implement its methods
    - Why? So that it can be extendable by any additional algorithm we may choose in the future
2. I chose to keep the design simple -- it's all in-memory, running on a single process
    - Why? Time constraint mostly, but also to build something more substantial I'd need more in-depth requirements!
3. No shared state between instances
    - Why? Mostly constrained by time, but also I'd then need to add some kind of shared state store like Redis and that's just unneeded complexity for this demo
4. "Hard-coded" configuration for rate limiter (configuration passed at creation time so it's not dynamic/reloadable)
    -  Why? I considered adding some kind of "config.go" for the rate limiter but since I only have one algorithm so far, it seemed like overengineering to create a separate config struct for it. I'd definitely add it if I were to implement more algorithms though, and also if we needed to change the rate limits without restarting the process


## Trade-offs 
- I considered implementing a full on Strategy pattern, something like this:
```go
type Strategy interface {
	Allow() bool
	Wait(ctx context.Context) error
}

type RateLimiter struct {
	strategy Strategy
}
```
- This would be useful if we wanted to allow for additional functionality for the actual rate limiter or if we wanted to dynamically switch the algorithm, but I didn't go with it because it's overkill and it's too overly engineered for something like this
- The Token Bucket algorithm is a good general purpose algorithm, but it's not the best use case everywhere. For example, if our system needed to maintain a steady throughput/stream of traffic, this wouldn't be an ideal algorithm to use since it's build around handling a burst of events/requests coming in


## Limitations
- Because I chose to use float64 for counting tokens and calculating the rate at which they're refilled to the bucket, there's a chance for rounding errors accumulating over time
    - I'm assuming it's fine to be a teensy bit off and sacrifice the utmost precision for this demo though
- The `Wait()` implementation isn't the most performant it could be
    - Currently it sleeps for the duration it will take to get a refill of the token, but race conditions could cause the token to be grabbed by another goroutine, so there could be wasted Wait loops
    - Multiple goroutines could be forced to compete for the same lock
    - Could be solved by signaling when a token is available, but due to the simplicity of the project it would be overkill I think
- No shared state between instances due to the time + complexity of implementing a shared state store
- No global limit of rate limiter instances due to the above point
- There's a single global mutex, which could become a problem under super heavy concurrency
- There's no metrics or monitoring since it's just a demo 


## Assumptions
- The rate limiter is fully independent, meaning that it doesn't actually keep track of specific users/ID's/IP addresses/any actual tracking of events/requests coming into your system. The assumption is that there would be some sort of BucketManager (or RateLimitManager) in your system that would keep track of identity/who the request is coming from tied to the individual rate limiter instance for that identity
- There will be one rate limiter per one user and it'll all be running on the same process
- This implementation hasn't been stress tested because it's not meant to be used in a production-grade environment 
