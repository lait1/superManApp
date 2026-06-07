package api

import (
	"sync"
	"time"
)

// rateLimiter is a lightweight in-memory token-bucket limiter keyed by an
// arbitrary string (here: the character id). It guards the /checkin endpoint
// against abuse as required by docs/09 §4 ("мягкий лимит на /checkin").
//
// It is intentionally simple (no external dependencies, stdlib only) and is
// safe for concurrent use. A background sweep is not run; stale buckets are
// refilled lazily on access, so memory grows only with the number of distinct
// active keys.
type rateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	capacity float64 // maximum tokens in a bucket
	refill   float64 // tokens added per second
	now      func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// newRateLimiter builds a limiter allowing bursts up to capacity, refilling at
// refillPerSec tokens/second.
func newRateLimiter(capacity, refillPerSec float64) *rateLimiter {
	return &rateLimiter{
		buckets:  make(map[string]*bucket),
		capacity: capacity,
		refill:   refillPerSec,
		now:      time.Now,
	}
}

// Allow consumes one token for key, returning true if the request is allowed.
func (rl *rateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &bucket{tokens: rl.capacity - 1, last: now}
		return true
	}

	// Refill based on elapsed time.
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * rl.refill
		if b.tokens > rl.capacity {
			b.tokens = rl.capacity
		}
		b.last = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
