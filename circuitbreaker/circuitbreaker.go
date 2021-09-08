package circuitbreaker

import (
	"errors"
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
