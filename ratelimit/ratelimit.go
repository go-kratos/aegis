package ratelimit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	// ErrLimitExceed is returned when the rate limiter is
	// triggered and the request is rejected due to limit exceeded.
	ErrLimitExceed = errors.New("rate limit exceeded")
)

// Ratelimit .
type Ratelimit interface {
	Allow(context.Context) (func(err error), error)
}

// Option is sre breaker option function
type Option func(*config)

// Config broker config.
type config struct {
}

// New .
func New(builder func() Ratelimit, opts ...Option) *Group {
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}
	g := &Group{
		new: builder,
		cfg: &cfg,
	}
	g.val.Store(make(map[string]Ratelimit))
	return g
}

// Group .
type Group struct {
	cfg   *config
	mutex sync.Mutex
	val   atomic.Value

	new func() Ratelimit
}

// Get .
func (g *Group) Get(name string) Ratelimit {
	v := g.val.Load().(map[string]Ratelimit)
	cb, ok := v[name]
	if ok {
		return cb
	}
	// slowpath for group don`t have specified name breaker.
	g.mutex.Lock()
	nv := make(map[string]Ratelimit, len(v)+1)
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
	limit := g.Get(name)
	done, err := limit.Allow(ctx)
	if err == nil {
		defer func() {
			if _, ok := err.(ignore); ok {
				done(nil)
				return
			}
			done(err)
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
