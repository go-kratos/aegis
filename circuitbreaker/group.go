package circuitbreaker

import (
	"github.com/go-kratos/aegis/internal/syncmap"
)

// CircuitBreakerFactory 定义一个闭包类型，用于创建 CircuitBreaker 实例
type CircuitBreakerFactory func() CircuitBreaker

// KeyCircuitBreaker is a circuit breaker that manages multiple circuit breakers by key.
type KeyCircuitBreaker struct {
	requests  syncmap.SyncMap[string, CircuitBreaker]
	cbFactory CircuitBreakerFactory
}

// NewGroupCircuitBreaker creates a new KeyCircuitBreaker with the given factory.
func NewGroupCircuitBreaker(factory CircuitBreakerFactory) *KeyCircuitBreaker {
	kcb := &KeyCircuitBreaker{
		cbFactory: factory,
	}
	return kcb
}

// GetCircuitBreaker returns a CircuitBreaker for the given key.
func (kcb *KeyCircuitBreaker) GetCircuitBreaker(key string) CircuitBreaker {
	cb, ok := kcb.requests.Load(key)
	if !ok {
		// 使用传入的闭包创建具体的 CircuitBreaker 实例
		cb, _ = kcb.requests.LoadOrStore(key, kcb.cbFactory())
	}
	return cb
}
