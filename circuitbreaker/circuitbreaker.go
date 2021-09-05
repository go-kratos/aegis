package circuitbreaker

import (
	"errors"
	"sync"
	"sync/atomic"
)

// ErrNotAllowed error not allowed.
var ErrNotAllowed = errors.New("circuitbreaker: not allowed for circuit open")

// State .
type State int

const (
	// StateOpen when circuit breaker open, request not allowed, after sleep
	// some duration, allow one single request for testing the health, if ok
	// then state reset to closed, if not continue the step.
	StateOpen State = iota
	// StateClosed when circuit breaker closed, request allowed, the breaker
	// calc the succeed ratio, if request num greater request setting and
	// ratio lower than the setting ratio, then reset state to open.
	StateClosed
	// StateHalfopen when circuit breaker open, after slepp some duration, allow
	// one request, but not state closed.
	StateHalfopen
)

// CircuitBreaker is a circuit breaker.
type CircuitBreaker interface {
	Allow() error
	MarkSuccess()
	MarkFailed()
}

// Group .
type Group struct {
	mutex sync.Mutex
	val   atomic.Value

	New func() CircuitBreaker
}

// Get .
func (g *Group) Get(name string) CircuitBreaker {
	v, ok := g.val.Load().(map[string]CircuitBreaker)
	if ok {
		cb, ok := v[name]
		if ok {
			return cb
		}
	}
	// slowpath for group don`t have specified name breaker.
	g.mutex.Lock()
	nv := make(map[string]CircuitBreaker, len(v)+1)
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
	cb := g.Get(name)
	err := cb.Allow()
	if err != nil {
		if err = fn(); err == nil {
			cb.MarkSuccess()
			return nil
		}
		switch v := err.(type) {
		case ignore:
			cb.MarkSuccess()
			err = v.error
		case drop:
			cb.MarkFailed()
			err = v.error
		default:
			cb.MarkFailed()
		}
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

type drop struct {
	error
}

// Drop drop the error.
func Drop(err error) error {
	return drop{err}
}
