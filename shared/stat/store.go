package stat

import "github.com/viant/gmetric/counter"

type store struct{}

func (p store) Keys() []string {
	return []string{
		ErrorKey,
		NoSuchKey,

		// Deprecated - Aerospike specific
		Timeout,
		// Deprecated - Aerospike specific
		Down,

		Canceled,
		DeadlineExceeded,
	}
}

func (p store) Map(value interface{}) int {
	if value == nil {
		return -1
	}
	switch val := value.(type) {
	case error:
		return 0
	case string:
		switch val {
		case NoSuchKey:
			return 1
		case Timeout:
			return 2
		case Down:
			return 3
		case Canceled:
			return 4
		case DeadlineExceeded:
			return 5
		}
	}
	return -1
}

func NewStore() counter.Provider {
	return &store{}
}
