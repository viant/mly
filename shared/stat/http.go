package stat

import "github.com/viant/gmetric/counter"

// TODO move to shared/client
type http struct{}

const (
	// Pending is the column for the in-flight gauge maintained by the
	// metric.EnterThenExit Inc/Dec pattern. The exporter publishes:
	//   - <op>_pending     -- the current-in-flight counter (defective; see below)
	//   - <op>_pending_Max -- per-bucket peak from the Occupancy CustomCounter
	//
	// Known defects in the underlying mechanism:
	//
	//   1. Bucket-mismatch on Exit. EnterThenExit captures the recent-bucket
	//      index at Enter time and decrements that same bucket on Exit. If
	//      the bucket has rotated between Enter and Exit, the wrong bucket
	//      is decremented -- the previous bucket's value goes negative
	//      while the current bucket's value drifts high.
	//
	//   2. Mutex serialization on Inc/Dec. The Dir typed value is not a
	//      string, so MultiCounter.incrementValueBy takes c.locker.Lock()
	//      on every Enter and Exit. Under high QPS this is a real
	//      serialization point.
	//
	// Defect #1 inflates both _pending (current) and _pending_Max (per-bucket
	// peak); the inflation is in the conservative direction (over-estimation),
	// so the metrics are still operationally useful in different regimes:
	//
	//   - _pending_Max grouped per reporting dimension (e.g. by
	//     availability_zone, environment, op): for high-QPS operations
	//     the per-group peak rises substantially above baseline noise
	//     during fleet-wide saturation events, making this the cleaner
	//     saturation signal for those operations.
	//
	//   - _pending summed across reporting instances: exhibits dramatic
	//     spikes during saturation for any QPS profile, partially
	//     amplified by defect #1. For low-QPS operations where the
	//     per-group _pending_Max signal is lost in baseline noise, the
	//     fleet sum is the more visible saturation signal.
	//
	// _pending_Max is the cleaner peak-concurrency signal for capacity
	// sizing; pick per-operation based on QPS profile.
	Pending = "pending"
	// Shed marks a request that the client did NOT send because the host's
	// circuit breaker was already in the down state when getHost() was
	// called. Distinct from Down, which marks the trip event itself
	// (the request that observed the connection error and called
	// FlagDown). Shed is the count of subsequent requests that the
	// breaker rejected before recovery.
	Shed = "shed"
)

func (p http) Keys() []string {
	// New keys must be appended at the end so existing column indices
	// remain stable for downstream consumers (Mimir queries, dashboards).
	return []string{
		ErrorKey,
		Pending,
		Down,
		Canceled,
		DeadlineExceeded,
		Shed,
	}
}

func (p http) Map(value interface{}) int {
	if value == nil {
		return -1
	}

	switch val := value.(type) {
	case error:
		return 0
	case string:
		switch val {
		case Pending:
			return 1
		case Down:
			return 2
		case Canceled:
			return 3
		case DeadlineExceeded:
			return 4
		case Shed:
			return 5
		}
	case Dir:
		return 1
	}

	return -1
}

func (p http) NewCounter() counter.CustomCounter {
	return new(Occupancy)
}

func NewHttp() counter.Provider {
	return http{}
}
