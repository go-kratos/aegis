package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	// ErrNotAllowed error not allowed.
	ErrNotAllowed = errors.New("circuitbreaker: not allowed for circuit open")
)

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

// Option is sre breaker option function
type Option func(*config)

// Config broker config.
type config struct {
}

// CircuitBreaker .
type CircuitBreaker interface {
	// if CircuitBreaker is open,ErrNotAllowed should be returned
	Allow(context.Context) error
	MarkSuccess()
	MarkFailed()
}

// New .
func New(builder func() CircuitBreaker, opts ...Option) *Group {
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}
	g := &Group{
		new: builder,
		cfg: &cfg,
	}
	g.val.Store(make(map[string]CircuitBreaker))
	return g
}

// Group .
type Group struct {
	cfg   *config
	mutex sync.Mutex
	val   atomic.Value

	new func() CircuitBreaker
}

// Get .
func (g *Group) Get(name string) CircuitBreaker {
	v := g.val.Load().(map[string]CircuitBreaker)
	cb, ok := v[name]
	if ok {
		return cb
	}
	// slowpath for group don`t have specified name breaker.
	g.mutex.Lock()
	nv := make(map[string]CircuitBreaker, len(v)+1)
	for i, j := range v {
		nv[i] = j
	}
	cb = g.new()
	nv[name] = cb
	g.val.Store(nv)
	g.mutex.Unlock()
	return cb
}

// Do runs your function in a synchronous manner, blocking until either your
// function succeeds or an error is returned, including circuit errors.
func (g *Group) Do(ctx context.Context, name string, fn func() error) error {
	cb := g.Get(name)
	err := cb.Allow(ctx)
	if err == nil {
		defer func() {
			if err == nil {
				cb.MarkSuccess()
				return
			}
			switch err.(type) {
			case ignore:
				cb.MarkSuccess()
				err = err.(ignore).error
			case drop:
				err = err.(drop).error
			default:
				cb.MarkFailed()
			}
		}()

		err = fn()
	}
	return err
}

type ignore struct {
	error
}

// Ignore .
func Ignore(err error) error {
	return ignore{err}
}

type drop struct {
	error
}

// Drop .
func Drop(err error) error {
	return drop{err}
}
