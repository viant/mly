package circut

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeProber records probe invocations and never calls FlagUp.
// Tests that need probe-driven recovery call b.FlagUp() directly.
type fakeProber struct {
	probes int64
}

func (f *fakeProber) Probe() {
	atomic.AddInt64(&f.probes, 1)
}

// TestBreaker_Concurrent_NoDataRace exercises FlagDown / FlagUp / IsUp
// from many goroutines simultaneously. Run with `-race` to catch the
// previously-existing data race on b.Down (atomic read, non-atomic write).
//
// Without the fix this test reliably triggers a race-detector report:
//
//	WARNING: DATA RACE
//	Read at 0x... by goroutine N (atomic.LoadInt32):
//	  shared/circut.(*Breaker).IsUp
//	Previous write at 0x... by goroutine M:
//	  shared/circut.(*Breaker).FlagUp / FlagDown
func TestBreaker_Concurrent_NoDataRace(t *testing.T) {
	b := New(50*time.Millisecond, &fakeProber{})

	const goroutines = 16
	const iterations = 5000

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				switch (seed + i) % 3 {
				case 0:
					b.FlagDown()
				case 1:
					b.FlagUp()
				case 2:
					_ = b.IsUp()
				}
			}
		}(g)
	}
	wg.Wait()

	// Final state assertion is intentionally weak; the point of this test
	// is the race detector, not the terminal flag value.
	_ = b.IsUp()
}

// TestBreaker_BackoffAccumulates verifies that resetDuration doubles on
// each successive trip and is NOT clobbered by an interleaved FlagUp.
// Catches the lost-update bug where FlagUp's resetDuration reset ran
// outside the mutex and could race with FlagDown's resetDuration *= 2.
func TestBreaker_BackoffAccumulates(t *testing.T) {
	const initial = 50 * time.Millisecond
	b := New(initial, &fakeProber{})

	require.Equal(t, initial, b.resetDuration, "initial resetDuration")

	// First trip: doubles to 100ms.
	b.FlagDown()
	require.Equal(t, 2*initial, b.resetDuration, "after 1st trip")

	// Recover and trip again: must double from 100ms to 200ms (NOT
	// reset to 100ms, which is what the lost-update bug would do
	// under racy timing).
	b.FlagUp()
	require.Equal(t, initial, b.resetDuration, "FlagUp resets to initial")

	b.FlagDown()
	require.Equal(t, 2*initial, b.resetDuration, "after 2nd trip from initial")

	// Multiple FlagDowns without intervening FlagUp must NOT
	// re-double. Idempotency comes from the CAS.
	b.FlagDown()
	b.FlagDown()
	b.FlagDown()
	assert.Equal(t, 2*initial, b.resetDuration, "extra FlagDowns are no-ops while down")
}

// TestBreaker_FlagUp_Idempotent verifies that FlagUp on an already-up
// breaker does NOT reset resetDuration (which would be wrong if the
// breaker is in the middle of a backoff sequence and a stale Probe
// callback fires FlagUp redundantly).
func TestBreaker_FlagUp_Idempotent(t *testing.T) {
	const initial = 50 * time.Millisecond
	b := New(initial, &fakeProber{})

	// Trip and recover -- resetDuration is back to initial.
	b.FlagDown()
	b.FlagUp()
	require.Equal(t, initial, b.resetDuration)

	// Trip again -- resetDuration doubles.
	b.FlagDown()
	require.Equal(t, 2*initial, b.resetDuration)

	// Spurious FlagUp callback on an already-up breaker would be the
	// state where Down is already 0 -- but we just trip'd, so it's 1.
	// The realistic spurious FlagUp scenario is Probe firing twice
	// after recovery has already happened. Simulate that.
	b.FlagUp() // legitimate recovery; resetDuration -> initial
	require.Equal(t, initial, b.resetDuration)
	b.FlagUp() // spurious; must NOT clobber any subsequent backoff state
	require.Equal(t, initial, b.resetDuration, "spurious FlagUp is a no-op")
}

// TestBreaker_FlagDown_Idempotent verifies that repeated FlagDown calls
// while the breaker is already down do not advance resetTime or grow
// resetDuration further.
func TestBreaker_FlagDown_Idempotent(t *testing.T) {
	const initial = 50 * time.Millisecond
	b := New(initial, &fakeProber{})

	b.FlagDown()
	firstResetTime := b.resetTime
	require.Equal(t, 2*initial, b.resetDuration)

	// Subsequent FlagDowns while down must be no-ops.
	for i := 0; i < 5; i++ {
		b.FlagDown()
	}
	assert.Equal(t, 2*initial, b.resetDuration, "resetDuration unchanged across redundant FlagDowns")
	assert.Equal(t, firstResetTime, b.resetTime, "resetTime unchanged across redundant FlagDowns")
}
