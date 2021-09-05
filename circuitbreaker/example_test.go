package circuitbreaker_test

import (
	"errors"
	"fmt"

	"github.com/go-kratos/aegis/circuitbreaker"
	"github.com/go-kratos/aegis/circuitbreaker/sre"
)

// This is a example of using a circuit breaker Do() when return nil.
func Example() {
	g := &circuitbreaker.Group{New: func() circuitbreaker.CircuitBreaker {
		return sre.NewBreaker()
	}}
	err := g.Do("do", func() error {
		// dosomething
		return nil
	})

	fmt.Printf("err=%v", err)
	// Output: err=<nil>
}

// This is a example of using a circuit breaker fn failed then call fallback.
func Example_fallback() {
	g := &circuitbreaker.Group{New: func() circuitbreaker.CircuitBreaker {
		return sre.NewBreaker()
	}}
	err := g.Do("do", func() error {
		// dosomething
		return errors.New("fallback")
	})

	fmt.Printf("err=%v", err)
	// Output: err=fallback
}

// This is a example of using a circuit breaker fn failed but ignore error mark
// as success.
func Example_ignore() {
	g := &circuitbreaker.Group{New: func() circuitbreaker.CircuitBreaker {
		return sre.NewBreaker()
	}}
	err := g.Do("do", func() error {
		// dosomething
		return circuitbreaker.Ignore(errors.New("ignore"))
	})

	fmt.Printf("err=%v", err)
	// Output: err=ignore
}
