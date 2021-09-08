package bbr

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/go-kratos/aegis/pkg/cpu"
	"github.com/go-kratos/aegis/pkg/window"
	"github.com/go-kratos/aegis/ratelimit"
)

var (
	gCPU  int64
	decay = 0.95

	_ ratelimit.Limiter = &BBR{}
)

type (
	cpuGetter func() int64

	// Option function for bbr limiter
	Option func(*options)
)

func init() {
	go cpuproc()
}

// cpu = cpuᵗ⁻¹ * decay + cpuᵗ * (1 - decay)
func cpuproc() {
	ticker := time.NewTicker(time.Millisecond * 500) // same to cpu sample rate
	defer func() {
		ticker.Stop()
		if err := recover(); err != nil {
			go cpuproc()
		}
	}()

	// EMA algorithm: https://blog.csdn.net/m0_38106113/article/details/81542863
	for range ticker.C {
		stat := &cpu.Stat{}
		cpu.ReadStat(stat)
		prevCPU := atomic.LoadInt64(&gCPU)
		curCPU := int64(float64(prevCPU)*decay + float64(stat.Usage)*(1.0-decay))
		atomic.StoreInt64(&gCPU, curCPU)
	}
}

// Stat contains the metrics snapshot of bbr.
type Stat struct {
	CPU         int64
	InFlight    int64
	MaxInFlight int64
	MinRt       int64
	MaxPass     int64
}

// counterCache is used to cache maxPASS and minRt result.
// Value of current bucket is not counted in real time.
// Cache time is equal to a bucket duration.
type counterCache struct {
	val  int64
	time time.Time
}

// options of bbr limiter.
type options struct {
	// WindowSize defines time duration per window
	Window time.Duration
	// BucketNum defines bucket number for each window
	Bucket int
	// CPUThreshold
	CPUThreshold int64
}

// WithWindow with window size.
func WithWindow(d time.Duration) Option {
	return func(o *options) {
		o.Window = d
	}
}

// WithBucket with bucket ize.
func WithBucket(b int) Option {
	return func(o *options) {
		o.Bucket = b
	}
}

// WithCPUThreshold with cpu threshold;
func WithCPUThreshold(threshold int64) Option {
	return func(o *options) {
		o.CPUThreshold = threshold
	}
}

// BBR implements bbr-like limiter.
// It is inspired by sentinel.
// https://github.com/alibaba/Sentinel/wiki/%E7%B3%BB%E7%BB%9F%E8%87%AA%E9%80%82%E5%BA%94%E9%99%90%E6%B5%81
type BBR struct {
	cpu             cpuGetter
	passStat        window.RollingCounter
	rtStat          window.RollingCounter
	inFlight        int64
	bucketPerSecond int64
	bucketDuration  time.Duration

	// prevDropTime defines previous start drop since initTime
	prevDropTime atomic.Value
	maxPASSCache atomic.Value
	minRtCache   atomic.Value

	opts options
}

// NewLimiter returns a bbr limiter
func NewLimiter(opts ...Option) *BBR {
	opt := options{
		Window:       time.Second * 10,
		Bucket:       100,
		CPUThreshold: 800,
	}
	for _, o := range opts {
		o(&opt)
	}

	bucketDuration := opt.Window / time.Duration(opt.Bucket)
	passStat := window.NewRollingCounter(window.RollingCounterOpts{Size: opt.Bucket, BucketDuration: bucketDuration})
	rtStat := window.NewRollingCounter(window.RollingCounterOpts{Size: opt.Bucket, BucketDuration: bucketDuration})

	limiter := &BBR{
		opts:            opt,
		passStat:        passStat,
		rtStat:          rtStat,
		bucketDuration:  bucketDuration,
		bucketPerSecond: int64(time.Second / bucketDuration),
		cpu:             func() int64 { return atomic.LoadInt64(&gCPU) },
	}
	return limiter
}

func (l *BBR) maxPASS() int64 {
	passCache := l.maxPASSCache.Load()
	if passCache != nil {
		ps := passCache.(*counterCache)
		if l.timespan(ps.time) < 1 {
			return ps.val
		}
	}
	rawMaxPass := int64(l.passStat.Reduce(func(iterator window.Iterator) float64 {
		var result = 1.0
		for i := 1; iterator.Next() && i < l.opts.Bucket; i++ {
			bucket := iterator.Bucket()
			count := 0.0
			for _, p := range bucket.Points {
				count += p
			}
			result = math.Max(result, count)
		}
		return result
	}))
	if rawMaxPass == 0 {
		rawMaxPass = 1
	}
	l.maxPASSCache.Store(&counterCache{
		val:  rawMaxPass,
		time: time.Now(),
	})
	return rawMaxPass
}

// timespan returns the passed bucket count
// since lastTime, if it is one bucket duration earlier than
// the last recorded time, it will return the BucketNum.
func (l *BBR) timespan(lastTime time.Time) int {
	v := int(time.Since(lastTime) / l.bucketDuration)
	if v > -1 {
		return v
	}
	return l.opts.Bucket
}

func (l *BBR) minRT() int64 {
	rtCache := l.minRtCache.Load()
	if rtCache != nil {
		rc := rtCache.(*counterCache)
		if l.timespan(rc.time) < 1 {
			return rc.val
		}
	}
	rawMinRT := int64(math.Ceil(l.rtStat.Reduce(func(iterator window.Iterator) float64 {
		var result = math.MaxFloat64
		for i := 1; iterator.Next() && i < l.opts.Bucket; i++ {
			bucket := iterator.Bucket()
			if len(bucket.Points) == 0 {
				continue
			}
			total := 0.0
			for _, p := range bucket.Points {
				total += p
			}
			avg := total / float64(bucket.Count)
			result = math.Min(result, avg)
		}
		return result
	})))
	if rawMinRT <= 0 {
		rawMinRT = 1
	}
	l.minRtCache.Store(&counterCache{
		val:  rawMinRT,
		time: time.Now(),
	})
	return rawMinRT
}

func (l *BBR) maxInFlight() int64 {
	return int64(math.Floor(float64(l.maxPASS()*l.minRT()*l.bucketPerSecond)/1000.0) + 0.5)
}

func (l *BBR) shouldDrop() bool {
	now := time.Duration(time.Now().UnixNano())
	if l.cpu() < l.opts.CPUThreshold {
		// current cpu payload below the threshold
		prevDropTime, _ := l.prevDropTime.Load().(time.Duration)
		if prevDropTime == 0 {
			// haven't start drop,
			// accept current request
			return false
		}
		if time.Duration(now-prevDropTime) <= time.Second {
			// just start drop one second ago,
			// check current inflight count
			inFlight := atomic.LoadInt64(&l.inFlight)
			return inFlight > 1 && inFlight > l.maxInFlight()
		}
		l.prevDropTime.Store(time.Duration(0))
		return false
	}
	// current cpu payload exceeds the threshold
	inFlight := atomic.LoadInt64(&l.inFlight)
	drop := inFlight > 1 && inFlight > l.maxInFlight()
	if drop {
		prevDrop, _ := l.prevDropTime.Load().(time.Duration)
		if prevDrop != 0 {
			// already started drop, return directly
			return drop
		}
		// store start drop time
		l.prevDropTime.Store(now)
	}
	return drop
}

// Stat tasks a snapshot of the bbr limiter.
func (l *BBR) Stat() Stat {
	return Stat{
		CPU:         l.cpu(),
		MinRt:       l.minRT(),
		MaxPass:     l.maxPASS(),
		MaxInFlight: l.maxInFlight(),
		InFlight:    atomic.LoadInt64(&l.inFlight),
	}
}

// Allow checks all inbound traffic.
// Once overload is detected, it raises limit.ErrLimitExceed error.
func (l *BBR) Allow() (ratelimit.DoneFunc, error) {
	if l.shouldDrop() {
		return nil, ratelimit.ErrLimitExceed
	}
	atomic.AddInt64(&l.inFlight, 1)
	start := time.Now().UnixNano()
	return func(ratelimit.DoneInfo) {
		rt := (time.Now().UnixNano() - start) / int64(time.Millisecond)
		l.rtStat.Add(rt)
		atomic.AddInt64(&l.inFlight, -1)
		l.passStat.Add(1)
	}, nil
}
