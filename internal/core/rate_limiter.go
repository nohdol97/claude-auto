package core

import (
	"strings"
	"sync"
	"time"
)

// RateLimiter manages rate limiting for Claude API calls
type RateLimiter struct {
	mu           sync.RWMutex
	limited      bool
	retryAfter   time.Time
	requests     []time.Time
	maxRequests  int
	window       time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		maxRequests: 10,
		window:      time.Minute,
		requests:    make([]time.Time, 0),
	}
}

// Wait waits if rate limited
func (rl *RateLimiter) Wait() error {
	rl.mu.RLock()
	if rl.limited && time.Now().Before(rl.retryAfter) {
		waitTime := time.Until(rl.retryAfter)
		rl.mu.RUnlock()
		time.Sleep(waitTime)
		return nil
	}
	rl.mu.RUnlock()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Clean old requests
	now := time.Now()
	cutoff := now.Add(-rl.window)
	newRequests := make([]time.Time, 0)
	for _, req := range rl.requests {
		if req.After(cutoff) {
			newRequests = append(newRequests, req)
		}
	}
	rl.requests = newRequests

	// Check if we're within limits
	if len(rl.requests) >= rl.maxRequests {
		rl.limited = true
		rl.retryAfter = rl.requests[0].Add(rl.window)
		waitTime := time.Until(rl.retryAfter)
		time.Sleep(waitTime)
		rl.limited = false
	}

	// Record this request
	rl.requests = append(rl.requests, now)
	return nil
}

// SetRateLimit sets rate limit status
func (rl *RateLimiter) SetRateLimit(retryAfter time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limited = true
	rl.retryAfter = time.Now().Add(retryAfter)
}

// IsRateLimited checks if currently rate limited
func (rl *RateLimiter) IsRateLimited() bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.limited && time.Now().Before(rl.retryAfter)
}

// GetRetryAfter returns the time when rate limit expires
func (rl *RateLimiter) GetRetryAfter() time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.retryAfter
}

// Reset resets the rate limiter
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limited = false
	rl.retryAfter = time.Time{}
	rl.requests = make([]time.Time, 0)
}

// IsRateLimitError checks if the output contains rate limit error patterns
func IsRateLimitError(output string) bool {
	lowerOutput := strings.ToLower(output)
	patterns := []string{
		"rate limit",
		"too many requests",
		"please wait",
		"retry after",
		"api rate limit",
		"quota exceeded",
		"throttled",
	}

	for _, pattern := range patterns {
		if strings.Contains(lowerOutput, pattern) {
			return true
		}
	}
	return false
}

// ParseRetryAfter attempts to parse retry-after duration from error message
func ParseRetryAfter(output string) time.Duration {
	// Default retry after 60 seconds if we can't parse
	defaultRetry := 60 * time.Second

	lowerOutput := strings.ToLower(output)

	// Try to find common patterns
	if strings.Contains(lowerOutput, "60 seconds") || strings.Contains(lowerOutput, "1 minute") {
		return 60 * time.Second
	}
	if strings.Contains(lowerOutput, "5 minutes") {
		return 5 * time.Minute
	}
	if strings.Contains(lowerOutput, "10 minutes") {
		return 10 * time.Minute
	}
	if strings.Contains(lowerOutput, "30 seconds") {
		return 30 * time.Second
	}

	return defaultRetry
}