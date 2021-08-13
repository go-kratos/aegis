package sre

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/sra/pkg/window"
)

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
	_mu   sync.RWMutex
	_conf = &Config{
		Window:  time.Duration(3 * time.Second),
		Bucket:  10,
		Request: 100,

		// Percentage of failures must be lower than 33.33%
		K: 1.5,

		// Pattern: "",
	}
	ErrBreakerTriggered = errors.New("circuit breaker triggered")
)

// Config broker config.
type Config struct {
	SwitchOff bool // breaker switch,default off.

	// Google
	K float64

	Window  time.Duration
	Bucket  int
	Request int64
}

func (conf *Config) fix() {
	if conf.K == 0 {
		conf.K = 1.5
	}
	if conf.Request == 0 {
		conf.Request = 100
	}
	if conf.Bucket == 0 {
		conf.Bucket = 10
	}
	if conf.Window == 0 {
		conf.Window = time.Duration(3 * time.Second)
	}
}

// sreBreaker is a sre CircuitBreaker pattern.
type sreBreaker struct {
	stat window.RollingCounter
	r    *rand.Rand
	// rand.New(...) returns a non thread safe object
	randLock sync.Mutex

	k       float64
	request int64

	state int32
}

func NewBreaker(c *Config) *sreBreaker {
	counterOpts := window.RollingCounterOpts{
		Size:           c.Bucket,
		BucketDuration: time.Duration(int64(c.Window) / int64(c.Bucket)),
	}
	stat := window.NewRollingCounter(counterOpts)
	return &sreBreaker{
		stat: stat,
		r:    rand.New(rand.NewSource(time.Now().UnixNano())),

		request: c.Request,
		k:       c.K,
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

func (b *sreBreaker) Allow() error {
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
