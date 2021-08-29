package sra

import "context"

// Warden is guardian
type Warden interface {
	// start protecting backend resources
	Ward(ctx context.Context, opts ...WardOption) (Done, error)
}

// Done is callback
type Done func(err error, opts ...DoneOption)

// WardOption is Ward Option
type WardOption func(*options)

// DoneOption is done Option
type DoneOption func(*doneOptions)

type options struct{}

type doneOptions struct{}
