package circut

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tinyWindow = 100 * time.Millisecond
	winK       = 3
)

func newTestLB(latest, rolling time.Duration, kConsecutive int, fraction float64) *LatencyBreaker {
	return NewLatencyBreaker(latest, rolling, tinyWindow, kConsecutive, fraction)
}

// TestLatencyBreaker_DisabledWhenZero verifies that a LatencyBreaker
// constructed with both thresholds = 0 is a no-op: Observe doesn't
// transition state, IsUp always returns true.
func TestLatencyBreaker_DisabledWhenZero(t *testing.T) {
	lb := newTestLB(0, 0, winK, 0.01)
	for i := 0; i < 10; i++ {
		lb.Observe(time.Hour)
	}
	assert.Equal(t, int32(0), lb.State())
	assert.True(t, lb.IsUp())
}

// TestLatencyBreaker_TripOnLatest verifies OFF -> ON transition when
// the latest observation alone exceeds LatestThreshold (rolling
// threshold disabled).
func TestLatencyBreaker_TripOnLatest(t *testing.T) {
	lb := newTestLB(40*time.Millisecond, 0, winK, 0.01)
	require.Equal(t, int32(0), lb.State())

	lb.Observe(50 * time.Millisecond)
	assert.Equal(t, int32(1), lb.State(), "single observation above latestThreshold trips ON")
}

// TestLatencyBreaker_TripOnRolling verifies OFF -> ON transition when
// the rolling average crosses RollingThreshold even though no single
// observation hits the latest threshold.
func TestLatencyBreaker_TripOnRolling(t *testing.T) {
	// LatestThreshold=0 (disabled), RollingThreshold=20ms.
	lb := newTestLB(0, 20*time.Millisecond, winK, 0.01)

	// Stream of 25ms observations -- each below latestThreshold (which
	// is disabled), but rolling crosses 20ms quickly.
	for i := 0; i < 5; i++ {
		lb.Observe(25 * time.Millisecond)
	}
	assert.Equal(t, int32(1), lb.State(), "sustained observations above rollingThreshold trip ON")
}

// TestLatencyBreaker_RecoverViaKConsecutive verifies ON -> OFF takes K
// consecutive observations satisfying the configured thresholds.
//
// Uses RollingThreshold=0 (disabled) so the test isolates the K-consecutive
// state machine from rolling-window pollution -- a single slow observation
// in the rolling window keeps the rolling average elevated for the full
// window duration regardless of how many subsequent fast observations
// arrive, which is correct production behavior but obscures this test's
// intent.
func TestLatencyBreaker_RecoverViaKConsecutive(t *testing.T) {
	lb := newTestLB(40*time.Millisecond, 0, winK, 1.0)

	// Trip via latest.
	lb.Observe(50 * time.Millisecond)
	require.Equal(t, int32(1), lb.State())

	// K-1 fast observations -- not enough to recover.
	for i := 0; i < winK-1; i++ {
		lb.Observe(5 * time.Millisecond)
		require.Equal(t, int32(1), lb.State(), "still ON after %d observations", i+1)
	}

	// Kth fast observation -- recovers.
	lb.Observe(5 * time.Millisecond)
	assert.Equal(t, int32(0), lb.State(), "ON -> OFF after K consecutive fast observations")
}

// TestLatencyBreaker_RecoveryResetsOnSlow verifies that a single
// above-threshold observation resets the consecutive-OK counter.
// RollingThreshold=0 to isolate the consecutive-OK reset logic.
func TestLatencyBreaker_RecoveryResetsOnSlow(t *testing.T) {
	lb := newTestLB(40*time.Millisecond, 0, winK, 1.0)

	lb.Observe(50 * time.Millisecond) // trip
	require.Equal(t, int32(1), lb.State())

	lb.Observe(5 * time.Millisecond) // 1 OK
	lb.Observe(5 * time.Millisecond) // 2 OK
	require.Equal(t, int32(1), lb.State(), "still ON before K consecutive")

	lb.Observe(50 * time.Millisecond) // bad observation -- resets consecutiveOK to 0
	require.Equal(t, int32(1), lb.State())

	// Now need K consecutive again from scratch.
	lb.Observe(5 * time.Millisecond)
	lb.Observe(5 * time.Millisecond)
	require.Equal(t, int32(1), lb.State(), "still ON after only 2 fast observations following reset")
	lb.Observe(5 * time.Millisecond)
	assert.Equal(t, int32(0), lb.State(), "ON -> OFF after K consecutive following reset")
}

// TestLatencyBreaker_PassThroughFraction verifies the probabilistic
// pass-through behavior while ON, using an injected deterministic
// random source.
func TestLatencyBreaker_PassThroughFraction(t *testing.T) {
	lb := newTestLB(40*time.Millisecond, 0, winK, 0.25)

	// Trip.
	lb.Observe(50 * time.Millisecond)
	require.Equal(t, int32(1), lb.State())

	// Inject a deterministic counter generating values 0.0, 0.1, 0.2,
	// 0.3, 0.4, ... -- pass-through fires when value < 0.25, i.e. for
	// the first 3 (0.0, 0.1, 0.2) of every 10.
	var counter int
	lb.randFloat = func() float64 {
		v := float64(counter%10) / 10.0
		counter++
		return v
	}

	pass := 0
	const N = 1000
	for i := 0; i < N; i++ {
		if lb.IsUp() {
			pass++
		}
	}
	// Expected: 30% pass exactly with this generator (3/10 buckets).
	// Allow ±2% drift for any rounding.
	assert.InDelta(t, 0.30, float64(pass)/float64(N), 0.02,
		"pass-through fraction should match injected random source")
}

// TestLatencyBreaker_PassThroughZero verifies that PassThroughFraction=0
// sheds 100% while ON.
func TestLatencyBreaker_PassThroughZero(t *testing.T) {
	lb := newTestLB(40*time.Millisecond, 0, winK, 0.0)
	lb.Observe(50 * time.Millisecond)
	require.Equal(t, int32(1), lb.State())

	for i := 0; i < 100; i++ {
		assert.False(t, lb.IsUp(), "PassThroughFraction=0 must shed 100%% while ON")
	}
}

// TestLatencyBreaker_NilSafe verifies a nil receiver behaves as
// a permanently-up no-op breaker (lets the caller treat
// "no LatencyBreaker configured" identically to "configured but OFF").
func TestLatencyBreaker_NilSafe(t *testing.T) {
	var lb *LatencyBreaker
	assert.True(t, lb.IsUp())
	lb.Observe(time.Second) // must not panic
	assert.Equal(t, int32(0), lb.State())
}

// TestLatencyBreaker_Concurrent_NoDataRace exercises Observe / IsUp
// from many goroutines simultaneously. Run with `-race` to catch any
// data races introduced by future edits.
func TestLatencyBreaker_Concurrent_NoDataRace(t *testing.T) {
	lb := newTestLB(40*time.Millisecond, 20*time.Millisecond, winK, 0.5)

	const goroutines = 16
	const iterations = 2000

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				switch (seed + i) % 3 {
				case 0:
					lb.Observe(10 * time.Millisecond)
				case 1:
					lb.Observe(50 * time.Millisecond)
				case 2:
					_ = lb.IsUp()
				}
			}
		}(g)
	}
	wg.Wait()

	// Sanity: state must be 0 or 1.
	st := lb.State()
	assert.True(t, st == 0 || st == 1, "state must be 0 or 1, got %d", st)
}

// TestRollingAverage_BucketRotation verifies that observations outside
// the rolling window are excluded from the average.
func TestRollingAverage_BucketRotation(t *testing.T) {
	r := newRollingAverage(100*time.Millisecond, 10)
	t0 := time.Unix(0, 1_000_000_000) // 1s exactly

	// Add 10 observations of 50ms within a single bucket period.
	for i := 0; i < 10; i++ {
		r.add(50*time.Millisecond, t0)
	}
	assert.Equal(t, 50*time.Millisecond, r.average(t0))

	// 200ms later, all old buckets should be outside the 100ms window.
	tLater := t0.Add(200 * time.Millisecond)
	assert.Equal(t, time.Duration(0), r.average(tLater),
		"window should expire all 10-bucket-old observations")
}

// Sanity check: ensure the atomic types we use compile.
var _ = atomic.LoadInt32
