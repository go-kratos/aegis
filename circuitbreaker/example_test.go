package circuitbreaker_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-kratos/sra/circuitbreaker"

	"github.com/go-kratos/sra/circuitbreaker/sre"
)

// This is a example of using a circuit breaker Do() when return nil.
func ExampleDo() {
	g := circuitbreaker.New(func() circuitbreaker.CircuitBreaker {
		return sre.NewBreaker()
	})
	err := g.Do(context.Background(), "do", func() error {
		// dosomething
		return nil
	})

	fmt.Printf("err=%v", err)
	// Output: err=<nil>
}

// This is a example of using a circuit breaker fn failed then call fallback.
func ExampleDo_fallback() {
	g := circuitbreaker.New(func() circuitbreaker.CircuitBreaker {
		return sre.NewBreaker()
	})
	err := g.Do(context.Background(), "do", func() error {
		// dosomething
		return errors.New("fallback")
	})

	fmt.Printf("err=%v", err)
	// Output: err=fallback
}

// This is a example of using a circuit breaker fn failed but ignore error mark
// as success.
func ExampleDo_ignore() {
	g := circuitbreaker.New(func() circuitbreaker.CircuitBreaker {
		return sre.NewBreaker()
	})
	err := g.Do(context.Background(), "do", func() error {
		// dosomething
		return circuitbreaker.Ignore(errors.New("fallback"))
	})

	fmt.Printf("err=%v", err)
	// Output: err=<nil>
}
