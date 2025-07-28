package circuitbreaker

import (
	"github.com/go-kratos/aegis/internal/syncmap"
)

// CircuitBreakerFactory 定义一个闭包类型，用于创建 CircuitBreaker 实例
type CircuitBreakerFactory func() CircuitBreaker

// Group is a circuit breaker that manages multiple circuit breakers by key.
type Group struct {
	requests  syncmap.SyncMap[string, CircuitBreaker]
	cbFactory CircuitBreakerFactory
}

// NewGroupCircuitBreaker creates a new Group with the given factory.
func NewGroup(factory CircuitBreakerFactory) *Group {
	g := &Group{
		cbFactory: factory,
	}
	return g
}

// GetCircuitBreaker returns a CircuitBreaker for the given key.
func (g *Group) GetCircuitBreaker(key string) CircuitBreaker {
	cb, ok := g.requests.Load(key)
	if !ok {
		// 使用传入的闭包创建具体的 CircuitBreaker 实例
		cb, _ = g.requests.LoadOrStore(key, g.cbFactory())
	}
	return cb
}
