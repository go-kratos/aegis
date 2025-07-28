package key

import (
	"time"

	"github.com/go-kratos/aegis/internal/syncmap"
	"golang.org/x/time/rate"
)

// Option is a function that configures the Limiter.
type Option func(*Limiter)

// WithExpires sets the expiration duration for the limiter.
func WithExpires(d time.Duration) Option {
	return func(l *Limiter) {
		l.expires = d
	}
}

// Limiter is a rate limiter that allows a certain number of requests per second.
type Limiter struct {
	burst    int
	interval time.Duration
	expires  time.Duration
	requests syncmap.SyncMap[string, *keyLimiter]
}

// NewLimiter creates a new RateLimiter with the given interval and burst size.
func NewLimiter(interval time.Duration, burst int, opts ...Option) *Limiter {
	l := &Limiter{
		burst:    burst,
		interval: interval,
		expires:  time.Minute,
	}
	for _, o := range opts {
		o(l)
	}
	go l.cleanupExpired()
	return l
}

// GetLimiter returns a Limiter for the given key.
func (l *Limiter) GetLimiter(key string) *rate.Limiter {
	limiter, ok := l.requests.Load(key)
	if !ok {
		limiter, ok = l.requests.LoadOrStore(key, &keyLimiter{
			Limiter: rate.NewLimiter(rate.Every(l.interval), l.burst),
		})
	}
	limiter.lastAccess = time.Now()
	return limiter.Limiter
}

func (l *Limiter) cleanupExpired() {
	ticker := time.NewTicker(l.expires)
	defer ticker.Stop()
	for range ticker.C {
		l.requests.Range(func(key string, value *keyLimiter) bool {
			if time.Since(value.lastAccess) > l.interval {
				l.requests.Delete(key)
			}
			return true
		})
	}
}

// keyLimiter is a rate limiter that does not allow any requests.
type keyLimiter struct {
	*rate.Limiter
	lastAccess time.Time
}
