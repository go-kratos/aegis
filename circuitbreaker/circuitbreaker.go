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
	m, ok := g.val.Load().(map[string]CircuitBreaker)
	if ok {
		breaker, ok := m[name]
		if ok {
			return breaker
		}
	}
	// slowpath for group don`t have specified name breaker.
	g.mutex.Lock()
	nm := make(map[string]CircuitBreaker, len(m)+1)
	for k, v := range m {
		nm[k] = v
	}
	breaker := g.New()
	nm[name] = breaker
	g.val.Store(nm)
	g.mutex.Unlock()
	return breaker
}

// Do runs your function in a synchronous manner, blocking until either your
// function succeeds or an error is returned, including circuit errors.
func (g *Group) Do(name string, fn func() error, fbs ...func(error) error) error {
	breaker := g.Get(name)
	err := breaker.Allow()
	if err == nil {
		if err = fn(); err == nil {
			breaker.MarkSuccess()
			return nil
		}
		switch v := err.(type) {
		case ignore:
			breaker.MarkSuccess()
			err = v.error
		case drop:
			breaker.MarkFailed()
			err = v.error
		default:
			breaker.MarkFailed()
		}
	}
	// fallback the request
	if err != nil {
		oe := err // save origin error
		for _, fb := range fbs {
			if err = fb(oe); err == nil {
				return nil
			}
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
