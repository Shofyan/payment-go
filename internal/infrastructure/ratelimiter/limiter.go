package ratelimiter

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter provides rate limiting functionality
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Wait(ctx context.Context, key string) error
}

// TokenBucketLimiter implements token bucket algorithm
// Each key (e.g., user ID, IP) gets its own bucket
type TokenBucketLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit // Tokens per second
	burst    int        // Bucket size
	cleanup  time.Duration
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
// rate: tokens per second (e.g., 100 means 100 requests per second)
// burst: maximum burst size (e.g., 200 means can burst up to 200 requests)
func NewTokenBucketLimiter(r float64, burst int) *TokenBucketLimiter {
	tbl := &TokenBucketLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(r),
		burst:    burst,
		cleanup:  5 * time.Minute,
	}

	// Start cleanup goroutine to prevent memory leak
	go tbl.cleanupLoop()

	return tbl
}

// Allow checks if request is allowed (non-blocking)
func (tbl *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	limiter := tbl.getLimiter(key)
	return limiter.Allow(), nil
}

// Wait blocks until request is allowed or context is cancelled
func (tbl *TokenBucketLimiter) Wait(ctx context.Context, key string) error {
	limiter := tbl.getLimiter(key)
	return limiter.Wait(ctx)
}

// AllowN checks if N tokens are available
func (tbl *TokenBucketLimiter) AllowN(key string, n int) bool {
	limiter := tbl.getLimiter(key)
	return limiter.AllowN(time.Now(), n)
}

// getLimiter returns or creates a limiter for a key
func (tbl *TokenBucketLimiter) getLimiter(key string) *rate.Limiter {
	tbl.mu.RLock()
	limiter, exists := tbl.limiters[key]
	tbl.mu.RUnlock()

	if exists {
		return limiter
	}

	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	// Double-check after acquiring write lock
	limiter, exists = tbl.limiters[key]
	if exists {
		return limiter
	}

	// Create new limiter
	limiter = rate.NewLimiter(tbl.rate, tbl.burst)
	tbl.limiters[key] = limiter

	return limiter
}

// cleanupLoop removes idle limiters to prevent memory leak
func (tbl *TokenBucketLimiter) cleanupLoop() {
	ticker := time.NewTicker(tbl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		tbl.mu.Lock()
		// Clear all limiters periodically to prevent memory leak
		// In production, this would track last access time and only remove idle ones
		tbl.limiters = make(map[string]*rate.Limiter)
		tbl.mu.Unlock()
	}
}

// LeakyBucketLimiter implements leaky bucket algorithm
// Requests leak out at a constant rate
type LeakyBucketLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	capacity int           // Bucket capacity
	leakRate time.Duration // Time to leak one request
}

type bucket struct {
	tokens   int
	lastLeak time.Time
	mu       sync.Mutex
}

// NewLeakyBucketLimiter creates a new leaky bucket rate limiter
func NewLeakyBucketLimiter(capacity int, leakRate time.Duration) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		buckets:  make(map[string]*bucket),
		capacity: capacity,
		leakRate: leakRate,
	}
}

// Allow checks if request is allowed
func (lbl *LeakyBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	b := lbl.getBucket(key)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Leak tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(b.lastLeak)
	tokensToLeak := int(elapsed / lbl.leakRate)

	if tokensToLeak > 0 {
		b.tokens -= tokensToLeak
		if b.tokens < 0 {
			b.tokens = 0
		}
		b.lastLeak = now
	}

	// Check if we can add new token
	if b.tokens < lbl.capacity {
		b.tokens++
		return true, nil
	}

	return false, nil
}

// Wait is not implemented for leaky bucket (blocks indefinitely)
func (lbl *LeakyBucketLimiter) Wait(ctx context.Context, key string) error {
	return errors.New("wait not implemented for leaky bucket")
}

func (lbl *LeakyBucketLimiter) getBucket(key string) *bucket {
	lbl.mu.RLock()
	b, exists := lbl.buckets[key]
	lbl.mu.RUnlock()

	if exists {
		return b
	}

	lbl.mu.Lock()
	defer lbl.mu.Unlock()

	// Double-check
	b, exists = lbl.buckets[key]
	if exists {
		return b
	}

	b = &bucket{
		tokens:   0,
		lastLeak: time.Now(),
	}
	lbl.buckets[key] = b

	return b
}

// SlidingWindowLimiter implements sliding window counter algorithm
// More accurate than fixed window, prevents burst at window boundaries
type SlidingWindowLimiter struct {
	mu      sync.RWMutex
	windows map[string]*window
	limit   int
	window  time.Duration
}

type window struct {
	requests []time.Time
	mu       sync.Mutex
}

// NewSlidingWindowLimiter creates a sliding window rate limiter
func NewSlidingWindowLimiter(limit int, windowDuration time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windows: make(map[string]*window),
		limit:   limit,
		window:  windowDuration,
	}
}

// Allow checks if request is allowed
func (swl *SlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
	w := swl.getWindow(key)

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-swl.window)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0, len(w.requests))
	for _, reqTime := range w.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	w.requests = validRequests

	// Check if limit exceeded
	if len(w.requests) >= swl.limit {
		return false, nil
	}

	// Add new request
	w.requests = append(w.requests, now)
	return true, nil
}

// Wait is not efficiently implemented for sliding window
func (swl *SlidingWindowLimiter) Wait(ctx context.Context, key string) error {
	return errors.New("wait not efficiently implemented for sliding window")
}

func (swl *SlidingWindowLimiter) getWindow(key string) *window {
	swl.mu.RLock()
	w, exists := swl.windows[key]
	swl.mu.RUnlock()

	if exists {
		return w
	}

	swl.mu.Lock()
	defer swl.mu.Unlock()

	w, exists = swl.windows[key]
	if exists {
		return w
	}

	w = &window{
		requests: make([]time.Time, 0, swl.limit),
	}
	swl.windows[key] = w

	return w
}

// RateLimitError represents rate limit exceeded error
type RateLimitError struct {
	Key        string
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return "rate limit exceeded"
}
