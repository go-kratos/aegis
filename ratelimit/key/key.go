package key

import (
	"sync"
	"time"

	"github.com/go-kratos/aegis/internal/syncmap"
	"golang.org/x/time/rate"
)

// Limiter is a rate limiter that allows a certain number of requests per second.
type Limiter struct {
	mu       sync.Mutex
	burst    int
	interval time.Duration
	requests syncmap.SyncMap[string, *keyLimiter]
}

// NewLimiter creates a new RateLimiter with the given interval and burst size.
func NewLimiter(interval time.Duration, burst int) *Limiter {
	l := &Limiter{
		burst:    burst,
		interval: interval,
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
	ticker := time.NewTicker(l.interval)
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
