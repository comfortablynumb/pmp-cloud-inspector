package ratelimit

import (
	"context"
	"time"
)

// Limiter provides rate limiting functionality for API calls
type Limiter struct {
	delay time.Duration
}

// New creates a new rate limiter with the specified delay between calls
func New(delay time.Duration) *Limiter {
	return &Limiter{
		delay: delay,
	}
}

// NewFromMilliseconds creates a rate limiter with delay specified in milliseconds
func NewFromMilliseconds(ms int) *Limiter {
	return &Limiter{
		delay: time.Duration(ms) * time.Millisecond,
	}
}

// Wait pauses for the configured delay duration
// Returns early if context is canceled
func (l *Limiter) Wait(ctx context.Context) error {
	if l.delay <= 0 {
		return nil
	}

	select {
	case <-time.After(l.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Delay returns the configured delay duration
func (l *Limiter) Delay() time.Duration {
	return l.delay
}
