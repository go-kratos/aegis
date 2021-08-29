package sre

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/sra"
	"github.com/go-kratos/sra/pkg/window"
)

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
	ErrBreakerTriggered = errors.New("circuit breaker triggered")
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

// WithBuckerNumber set the bucket number in a window duration
func WithBucketNumber(num int) Option {
	return func(c *config) {
		c.bucket = num
	}
}

// sreBreaker is a sre CircuitBreaker pattern.
type sreBreaker struct {
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

func NewBreaker(opts ...Option) *sreBreaker {
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
	return &sreBreaker{
		stat: stat,
		r:    rand.New(rand.NewSource(time.Now().UnixNano())),

		request: c.request,
		k:       c.k,
		state:   StateClosed,
	}
}

func (b *sreBreaker) summary() (success int64, total int64) {
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

func (b *sreBreaker) Allow(_ context.Context) error {
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
		return ErrBreakerTriggered
	}
	return nil
}

func (b *sreBreaker) MarkSuccess() {
	b.stat.Add(1)
}

func (b *sreBreaker) MarkFailed() {
	// NOTE: when client reject requets locally, continue add counter let the
	// drop ratio higher.
	b.stat.Add(0)
}

func (b *sreBreaker) trueOnProba(proba float64) (truth bool) {
	b.randLock.Lock()
	truth = b.r.Float64() < proba
	b.randLock.Unlock()
	return
}

func (b *sreBreaker) Check(err error) bool {
	return err == nil
}

func (b *sreBreaker) Mark(isSuccess bool) {
	if isSuccess {
		b.stat.Add(1)
	} else {
		b.stat.Add(0)
	}
}

func (b *sreBreaker) Ward(ctx context.Context, opts ...sra.WardOption) (sra.Done, error) {
	err := b.Allow(ctx)
	if err != nil {
		return nil, err
	}
	return func(e error, opts ...sra.DoneOption) {
		if e != nil {
			b.Mark(false)
		} else {
			b.Mark(true)
		}
	}, nil
}
