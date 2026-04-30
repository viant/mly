package circut

import (
	"sync"
	"sync/atomic"
	"time"
)

// Breaker helps manage back-off in case of a resource checked by prober is unavailable.
type Breaker struct {
	prober               Prober
	Down                 int32
	mux                  sync.RWMutex
	resetTime            time.Time
	resetDuration        time.Duration
	initialResetDuration time.Duration
}

// IsUp returns true if resource is up, and will trigger a probe if it is due.
func (b *Breaker) IsUp() bool {
	isUp := atomic.LoadInt32(&b.Down) == 0
	if !isUp {
		b.resetIfDue()
	}
	return isUp
}

// FlagUp is used to reset the backoff.
//
// Uses CompareAndSwap so the resetDuration reset only fires on an actual
// down->up transition (not on idempotent FlagUp calls), and so the write
// to b.Down is atomic with respect to IsUp's atomic.LoadInt32. The
// resetDuration write is performed under the mutex so it cannot race
// with FlagDown's resetDuration *= 2 (lost-update bug).
func (b *Breaker) FlagUp() {
	if !atomic.CompareAndSwapInt32(&b.Down, 1, 0) {
		return
	}
	b.mux.Lock()
	b.resetDuration = b.initialResetDuration
	b.mux.Unlock()
}

// resetIfDue will spawn a goroutine to probe the resource if the backoff time
// has passed.
func (b *Breaker) resetIfDue() {
	b.mux.RLock()
	dueTime := time.Now().After(b.resetTime)
	b.mux.RUnlock()
	if !dueTime {
		return
	}

	b.mux.Lock()
	dueTime = time.Now().After(b.resetTime)
	if !dueTime {
		b.mux.Unlock()
		return
	}
	b.resetTime = time.Now().Add(b.resetDuration)
	b.resetDuration = time.Duration(float32(b.resetDuration) * 1.5)
	b.mux.Unlock()

	go b.prober.Probe()
}

// FlagDown is used to indicate the resource is down.
//
// CompareAndSwap atomically transitions the Down flag exactly once per
// up->down edge, so backoff state (resetTime, resetDuration) is updated
// once per trip even under concurrent FlagDown calls. The atomic write
// is also synchronized with IsUp's atomic.LoadInt32.
func (b *Breaker) FlagDown() {
	if !atomic.CompareAndSwapInt32(&b.Down, 0, 1) {
		return
	}
	b.mux.Lock()
	b.resetTime = time.Now().Add(b.resetDuration)
	b.resetDuration *= 2 //double reset time each time service is Down
	b.mux.Unlock()
}

// New creates a new circut breaker
func New(resetDuration time.Duration, prober Prober) *Breaker {
	return &Breaker{
		prober:               prober,
		resetDuration:        resetDuration,
		initialResetDuration: resetDuration,
	}
}
