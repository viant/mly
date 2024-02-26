package stat

import "github.com/viant/gmetric/counter"

type client struct{}

const (
	EarlyCtxError = "ctx"
)

func (p client) Keys() []string {
	return []string{
		ErrorKey,
		NoSuchKey,
		Timeout,
		Down,
		Canceled,
		DeadlineExceeded,
		EarlyCtxError,
	}
}

func (p client) Map(value interface{}) int {
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
		case EarlyCtxError:
			return 6
		}
	}
	return -1
}

func NewClient() counter.Provider {
	return &client{}
}
