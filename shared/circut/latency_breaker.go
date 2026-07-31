package circut

import (
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// LatencyBreaker is a state machine that sheds traffic when observed
// request latencies exceed configured thresholds. It is independent of
// (and parallel to) the connection-failure-based Breaker.
//
// Detection:
//
//   - latest: the most recent observation. Compared against
//     LatestThreshold.
//   - rolling: average over a sliding window. Compared against
//     RollingThreshold.
//
// State transitions:
//
//   - OFF -> ON  on every observation where
//     latest > LatestThreshold OR rolling > RollingThreshold
//     (zero-valued thresholds are skipped, so a single threshold
//     can be used by leaving the other zero).
//   - ON  -> OFF after K consecutive observations satisfy
//     latest < LatestThreshold AND rolling < RollingThreshold.
//
// While ON, IsUp() returns true with probability PassThroughFraction
// and false otherwise -- letting a small fraction of traffic through
// to drive recovery sensing without committing real load.
//
// Concurrency:
//
// State (Down) is read with atomic.LoadInt32 in IsUp() and updated
// via atomic.CompareAndSwapInt32 from Observe(). Compound state
// (latest / rolling buckets / consecutiveOK) is protected by a
// mutex; only one Observe can mutate at a time. IsUp does not block.
//
// Random number source for pass-through is math/rand/v2 top-level,
// which is concurrent-safe and lock-free in Go 1.22+. A test seam
// (randFloat) allows deterministic tests.
type LatencyBreaker struct {
	// Configuration. Set at construction; not mutated after.
	LatestThreshold     time.Duration
	RollingThreshold    time.Duration
	RollingWindow       time.Duration
	KConsecutive        int
	PassThroughFraction float64

	// state holds the OFF (0) / ON (1) flag. Atomic.
	state int32

	mu            sync.Mutex // guards latest, rolling, consecutiveOK
	latest        time.Duration
	rolling       *rollingAverage
	consecutiveOK int

	// randFloat returns a value in [0, 1). Defaults to math/rand/v2.Float64.
	// Override for deterministic tests.
	randFloat func() float64
}

// NewLatencyBreaker constructs a LatencyBreaker. Zero-valued thresholds
// disable that branch of the trip predicate. If both thresholds are
// zero, Observe is a no-op and IsUp always returns true (effectively
// disabled).
func NewLatencyBreaker(
	latestThreshold, rollingThreshold, rollingWindow time.Duration,
	kConsecutive int,
	passThroughFraction float64,
) *LatencyBreaker {
	if rollingWindow <= 0 {
		rollingWindow = time.Second
	}
	if kConsecutive < 1 {
		kConsecutive = 1
	}
	if passThroughFraction < 0 {
		passThroughFraction = 0
	}
	if passThroughFraction > 1 {
		passThroughFraction = 1
	}
	return &LatencyBreaker{
		LatestThreshold:     latestThreshold,
		RollingThreshold:    rollingThreshold,
		RollingWindow:       rollingWindow,
		KConsecutive:        kConsecutive,
		PassThroughFraction: passThroughFraction,
		rolling:             newRollingAverage(rollingWindow, 10),
		randFloat:           rand.Float64,
	}
}

// IsUp returns true if the breaker is OFF (allowing all traffic), or
// true with probability PassThroughFraction if ON (allowing a small
// fraction through for recovery sensing).
func (lb *LatencyBreaker) IsUp() bool {
	if lb == nil {
		return true
	}
	if atomic.LoadInt32(&lb.state) == 0 {
		return true
	}
	return lb.randFloat() < lb.PassThroughFraction
}

// Observe records the latency of a completed request and advances the
// state machine. Called from the bidder client after each httpPost
// attempt completes (success or failure -- timeouts and errors count
// as observations and the elapsed time captured by the caller).
func (lb *LatencyBreaker) Observe(latency time.Duration) {
	if lb == nil {
		return
	}
	if lb.LatestThreshold == 0 && lb.RollingThreshold == 0 {
		// Both thresholds disabled; no signal to act on.
		return
	}

	lb.mu.Lock()
	now := time.Now()
	lb.latest = latency
	lb.rolling.add(latency, now)
	rollingAvg := lb.rolling.average(now)

	state := atomic.LoadInt32(&lb.state)

	// triggerOn: ANY threshold breached. Zero-valued thresholds skip.
	triggerOn := false
	if lb.LatestThreshold > 0 && latency > lb.LatestThreshold {
		triggerOn = true
	}
	if lb.RollingThreshold > 0 && rollingAvg > lb.RollingThreshold {
		triggerOn = true
	}

	// triggerOffReady: BOTH thresholds satisfied as below. Zero-valued
	// thresholds count as satisfied.
	triggerOffReady := true
	if lb.LatestThreshold > 0 && latency >= lb.LatestThreshold {
		triggerOffReady = false
	}
	if lb.RollingThreshold > 0 && rollingAvg >= lb.RollingThreshold {
		triggerOffReady = false
	}

	switch state {
	case 0: // OFF
		if triggerOn {
			atomic.StoreInt32(&lb.state, 1)
			lb.consecutiveOK = 0
		}
	case 1: // ON
		if triggerOffReady {
			lb.consecutiveOK++
			if lb.consecutiveOK >= lb.KConsecutive {
				atomic.StoreInt32(&lb.state, 0)
				lb.consecutiveOK = 0
			}
		} else {
			lb.consecutiveOK = 0
		}
	}
	lb.mu.Unlock()
}

// State returns 0 (OFF / up) or 1 (ON / shedding). Primarily for tests.
func (lb *LatencyBreaker) State() int32 {
	if lb == nil {
		return 0
	}
	return atomic.LoadInt32(&lb.state)
}

// rollingAverage keeps a sliding-window average of durations using a
// fixed number of time-aligned buckets. Buckets that fall outside the
// current window are reset on next access. All access is serialized
// by LatencyBreaker.mu; this struct is not goroutine-safe on its own.
type rollingAverage struct {
	window     time.Duration
	bucketDur  time.Duration
	bucketDurN int64 // bucketDur in nanoseconds, cached
	buckets    []rollingBucket
}

type rollingBucket struct {
	sum   time.Duration
	count int64
	until int64 // exclusive end of bucket period, in nanoseconds since epoch
}

func newRollingAverage(window time.Duration, n int) *rollingAverage {
	if n < 1 {
		n = 1
	}
	bd := window / time.Duration(n)
	if bd <= 0 {
		bd = window
		n = 1
	}
	return &rollingAverage{
		window:     window,
		bucketDur:  bd,
		bucketDurN: int64(bd),
		buckets:    make([]rollingBucket, n),
	}
}

// add records a value with completion time t.
func (r *rollingAverage) add(v time.Duration, t time.Time) {
	tn := t.UnixNano()
	idx := int((tn / r.bucketDurN) % int64(len(r.buckets)))
	until := ((tn / r.bucketDurN) + 1) * r.bucketDurN
	if r.buckets[idx].until != until {
		// Bucket belongs to a different period; reset and reuse.
		r.buckets[idx].sum = 0
		r.buckets[idx].count = 0
		r.buckets[idx].until = until
	}
	r.buckets[idx].sum += v
	r.buckets[idx].count++
}

// average returns the average across all buckets whose period overlaps
// the window ending at t. Returns 0 if no in-window samples.
func (r *rollingAverage) average(t time.Time) time.Duration {
	cutoff := t.UnixNano() - int64(r.window)
	var sum time.Duration
	var count int64
	for i := range r.buckets {
		b := &r.buckets[i]
		// Bucket's period end (until) must be after cutoff to be in-window.
		if b.until <= cutoff {
			continue
		}
		sum += b.sum
		count += b.count
	}
	if count == 0 {
		return 0
	}
	return sum / time.Duration(count)
}
