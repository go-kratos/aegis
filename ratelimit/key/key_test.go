package key

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func BenchmarkLimiter(b *testing.B) {
	// Create a new rate limiter with a limit of 1 request per second
	l := NewLimiter(rate.Every(time.Second), 1)

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("test_key_%d", i)
		limiter := l.GetLimiter(key)
		if !limiter.Allow() {
			b.Error("Expected request to be allowed")
		}
	}
}

func TestLimiter(t *testing.T) {
	// Create a new rate limiter with a limit of 1 request per second
	l := NewLimiter(rate.Every(time.Second), 1)

	limiter := l.GetLimiter("test_key")
	// Test that the first request is allowed
	if !limiter.Allow() {
		t.Error("Expected first request to be allowed")
	}

	// Test that the second request within the same second is not allowed
	if limiter.Allow() {
		t.Error("Expected second request to be denied")
	}

	// Wait for a second and test again
	time.Sleep(time.Second)

	// Test that the third request after waiting is allowed
	if !limiter.Allow() {
		t.Error("Expected third request to be allowed after waiting")
	}

	time.Sleep(time.Second)
	l.GetLimiter("test_ok")
	l.requests.Range(func(key string, value *keyLimiter) bool {
		if time.Since(value.lastAccess) > time.Second {
			return true
		}
		switch key {
		case "test_key":
			t.Errorf("Expected no requests for test_key after waiting, but found one")
		case "test_ok":
			if !value.Allow() {
				t.Error("Expected first request for test_key2 to be allowed")
			}
		default:
			t.Errorf("Unexpected key found: %s", key)
		}
		return true
	})
}
