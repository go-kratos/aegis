package circuitbreaker_test

import (
	"testing"

	"github.com/go-kratos/aegis/circuitbreaker"
	"github.com/go-kratos/aegis/circuitbreaker/sre"
	"github.com/stretchr/testify/assert"
)

func TestGroup_GetCircuitBreaker(t *testing.T) {
	g := circuitbreaker.NewGroup(func() circuitbreaker.CircuitBreaker {
		return sre.NewBreaker()
	})
	succ := g.GetCircuitBreaker("succ")
	assert.NotNil(t, succ)
	fail := g.GetCircuitBreaker("fail")
	assert.NotNil(t, fail)
	markSuccess(succ, 100)
	markFailed(fail, 10000)
	assert.Equal(t, succ.Allow(), nil)
	assert.NotEqual(t, fail.Allow(), nil)
}
func markSuccess(cb circuitbreaker.CircuitBreaker, count int) {
	for i := 0; i < count; i++ {
		cb.MarkSuccess()
	}
}
func markFailed(cb circuitbreaker.CircuitBreaker, count int) {
	for i := 0; i < count; i++ {
		cb.MarkFailed()
	}
}
