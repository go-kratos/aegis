package ratelimit

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	// ErrLimitExceed is returned when the rate limiter is
	// triggered and the request is rejected due to limit exceeded.
	ErrLimitExceed = errors.New("rate limit exceeded")
)

// Done is done function.
type Done func(DoneInfo)

// DoneInfo is done info.
type DoneInfo struct {
	err error
}

// Limiter is a rate limiter.
type Limiter interface {
	Allow() (Done, error)
}

// Group .
type Group struct {
	mutex sync.Mutex
	val   atomic.Value

	New func() Limiter
}

// Get .
func (g *Group) Get(name string) Limiter {
	m, ok := g.val.Load().(map[string]Limiter)
	if ok {
		limiter, ok := m[name]
		if ok {
			return limiter
		}
	}
	// slowpath for group don`t have specified name breaker.
	g.mutex.Lock()
	nm := make(map[string]Limiter, len(m)+1)
	for k, v := range m {
		nm[k] = v
	}
	limiter := g.New()
	nm[name] = limiter
	g.val.Store(nm)
	g.mutex.Unlock()
	return limiter
}

// Do runs your function in a synchronous manner, blocking until either your
// function succeeds or an error is returned, including circuit errors.
func (g *Group) Do(name string, fn func() error) error {
	limit := g.Get(name)
	done, err := limit.Allow()
	if err == nil {
		err = fn()
		if _, ok := err.(ignore); ok {
			err = nil
		}
		done(DoneInfo{err: err})
	}
	return err
}

type ignore struct {
	error
}

// Ignore ignore the error.
func Ignore(err error) error {
	return ignore{err}
}
