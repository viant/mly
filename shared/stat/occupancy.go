package stat

import (
	"sync"
	"sync/atomic"
)

type (
	// Occupancy keeps track of in-out metrics (not input-output); in-out counters are
	// numbers that track the number of start and end events; with a focus on tracking
	// the maximum number of starts without an end event.
	Occupancy struct {
		// Members are exposed due to how gmetrics operates.
		Max     uint64 `json:",omitempty"`
		current uint64
		l       sync.Mutex
	}

	// Dir (direction) should be true if this is an enter event; otherwise should be false.
	// It is expected that every Dir has an Exit, but that may not always happen.
	Dir bool
)

const (
	Enter = Dir(true)
	Exit  = Dir(false)
)

const minusOne = ^uint64(0)

// Implements counter.CustomCounter so that it can be passed to Occupancy, representing
// the element side of reduce.
func (_ Dir) Aggregate(value interface{}) {
	// does nothing
}

// Aggregate implements counter.CustomCounter, but represents the aggregate side.
func (a *Occupancy) Aggregate(value interface{}) {
	g, ok := value.(Dir)
	if !ok {
		return
	}

	var latest uint64
	if g {
		latest = atomic.AddUint64(&a.current, 1)
	} else {
		if a.current == 0 {
			return
		}

		latest = atomic.AddUint64(&a.current, minusOne)
		return
	}

	a.l.Lock()
	if latest > a.Max {
		a.Max = latest
	}
	a.l.Unlock()
}
