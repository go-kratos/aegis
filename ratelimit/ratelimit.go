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

// Limiter is a rate limiter.
type Limiter interface {
	Allow() (func(err error), error)
}

// Group .
type Group struct {
	mutex sync.Mutex
	val   atomic.Value

	New func() Limiter
}

// Get .
func (g *Group) Get(name string) Limiter {
	v, ok := g.val.Load().(map[string]Limiter)
	if ok {
		cb, ok := v[name]
		if ok {
			return cb
		}
	}
	// slowpath for group don`t have specified name breaker.
	g.mutex.Lock()
	nv := make(map[string]Limiter, len(v)+1)
	for i, j := range v {
		nv[i] = j
	}
	cb := g.New()
	nv[name] = cb
	g.val.Store(nv)
	g.mutex.Unlock()
	return cb
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
		done(err)
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
