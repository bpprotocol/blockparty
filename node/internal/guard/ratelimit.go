package guard

import (
	"sync"
	"time"
)

// RateLimiter is a per-source token-bucket limiter that bounds how much ingress
// work a single peer can drive. It is safe for concurrent use.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens added per second
	burst   float64 // bucket capacity
	now     func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter returns a limiter that refills ratePerSec tokens per second up
// to a capacity of burst tokens; each allowed call costs one token.
func NewRateLimiter(ratePerSec, burst float64) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    ratePerSec,
		burst:   burst,
		now:     time.Now,
	}
}

// Allow reports whether peer may perform one unit of work now, consuming a token.
func (r *RateLimiter) Allow(peer string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	b, ok := r.buckets[peer]
	if !ok {
		r.buckets[peer] = &bucket{tokens: r.burst - 1, last: now}
		return true
	}

	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(r.burst, b.tokens+elapsed*r.rate)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}
