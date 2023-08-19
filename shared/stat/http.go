package stat

import "github.com/viant/gmetric/counter"

type http struct{}

const Pending = "pending"

func (p http) Keys() []string {
	return []string{
		ErrorKey,
		Pending,
		Canceled,
		DeadlineExceeded,
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
		case Canceled:
			return 2
		case DeadlineExceeded:
			return 3
		}
	case *Occupancy:
		return 2
	}

	return -1
}

func (p http) CustomCounter() counter.CustomCounter {
	return new(Occupancy)
}

func NewHttp() counter.Provider {
	return http{}
}
