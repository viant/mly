package stat

import "github.com/viant/gmetric/counter"

// TODO move to shared/client
type http struct{}

const (
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
