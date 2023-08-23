package stat

import "github.com/viant/gmetric/counter"

type http struct{}

const Pending = "pending"

func (p http) Keys() []string {
	return []string{
		ErrorKey,
		Pending,
		Down,
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
		case Down:
			return 2
		case Canceled:
			return 3
		case DeadlineExceeded:
			return 4
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
