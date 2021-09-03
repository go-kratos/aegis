package sre

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/sra/circuitbreaker"
	"github.com/go-kratos/sra/pkg/window"
)

// Option is sre breaker option function
type Option func(*config)

const (
	// StateOpen when circuit breaker open, request not allowed, after sleep
	// some duration, allow one single request for testing the health, if ok
	// then state reset to closed, if not continue the step.
	StateOpen int32 = iota
	// StateClosed when circuit breaker closed, request allowed, the breaker
	// calc the succeed ratio, if request num greater request setting and
	// ratio lower than the setting ratio, then reset state to open.
	StateClosed
)

var (
	_ circuitbreaker.CircuitBreaker = &Breaker{}
)

// Config broker config.
type config struct {
	k float64

	window  time.Duration
	bucket  int
	request int64
}

// WithKValue set the K value of sre breaker, default K is 1.5
// Reducing the K will make adaptive throttling behave more aggressively,
// Increasing the K will make adaptive throttling behave less aggressively.
func WithKValue(K float64) Option {
	return func(c *config) {
		c.k = K
	}
}

// WithMinimumRequest set the minimum number of requests allowed
func WithMinimumRequest(request int64) Option {
	return func(c *config) {
		c.request = request
	}
}

// WithWindowSize set the duration size of the statistical window
func WithWindowSize(size time.Duration) Option {
	return func(c *config) {
		c.window = size
	}
}

// WithBucketNumber set the bucket number in a window duration
func WithBucketNumber(num int) Option {
	return func(c *config) {
		c.bucket = num
	}
}

// Breaker is a sre CircuitBreaker pattern.
type Breaker struct {
	stat window.RollingCounter
	r    *rand.Rand
	// rand.New(...) returns a non thread safe object
	randLock sync.Mutex

	// Reducing the k will make adaptive throttling behave more aggressively,
	// Increasing the k will make adaptive throttling behave less aggressively.
	k       float64
	request int64

	state int32
}

// NewBreaker return a sreBresker with options
func NewBreaker(opts ...Option) *Breaker {
	c := &config{
		k:       1.5,
		request: 100,
		bucket:  10,
		window:  3 * time.Second,
	}

	for _, o := range opts {
		o(c)
	}

	counterOpts := window.RollingCounterOpts{
		Size:           c.bucket,
		BucketDuration: time.Duration(int64(c.window) / int64(c.bucket)),
	}
	stat := window.NewRollingCounter(counterOpts)
	return &Breaker{
		stat: stat,
		r:    rand.New(rand.NewSource(time.Now().UnixNano())),

		request: c.request,
		k:       c.k,
		state:   StateClosed,
	}
}

func (b *Breaker) summary() (success int64, total int64) {
	b.stat.Reduce(func(iterator window.Iterator) float64 {
		for iterator.Next() {
			bucket := iterator.Bucket()
			total += bucket.Count
			for _, p := range bucket.Points {
				success += int64(p)
			}
		}
		return 0
	})
	return
}

// Allow request if error returns nil
func (b *Breaker) Allow(context.Context) error {
	success, total := b.summary()
	k := b.k * float64(success)

	// check overflow requests = K * success
	if total < b.request || float64(total) < k {
		if atomic.LoadInt32(&b.state) == StateOpen {
			atomic.CompareAndSwapInt32(&b.state, StateOpen, StateClosed)
		}
		return nil
	}
	if atomic.LoadInt32(&b.state) == StateClosed {
		atomic.CompareAndSwapInt32(&b.state, StateClosed, StateOpen)
	}
	dr := math.Max(0, (float64(total)-k)/float64(total+1))
	drop := b.trueOnProba(dr)

	if drop {
		return circuitbreaker.ErrNotAllowed
	}
	return nil
}

// MarkSuccess mark requeest is success
func (b *Breaker) MarkSuccess() {
	b.stat.Add(1)
}

// MarkFailed mark request is failed
func (b *Breaker) MarkFailed() {
	// NOTE: when client reject requets locally, continue add counter let the
	// drop ratio higher.
	b.stat.Add(0)
}

func (b *Breaker) trueOnProba(proba float64) (truth bool) {
	b.randLock.Lock()
	truth = b.r.Float64() < proba
	b.randLock.Unlock()
	return
}

// Check err if request is success
func (b *Breaker) Check(err error) bool {
	return err == nil
}

// Mark request
func (b *Breaker) Mark(isSuccess bool) {
	if isSuccess {
		b.stat.Add(1)
	} else {
		b.stat.Add(0)
	}
}
