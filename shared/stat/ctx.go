package stat

import "github.com/viant/gmetric/counter"

type ctxErrOnly struct{}

func (p ctxErrOnly) Keys() []string {
	return []string{
		ErrorKey,
		Canceled,
		DeadlineExceeded,
	}
}

func (p ctxErrOnly) Map(value interface{}) int {
	if value == nil {
		return -1
	}
	switch val := value.(type) {
	case error:
		return 0
	case string:
		switch val {
		case Canceled:
			return 1
		case DeadlineExceeded:
			return 2
		}
	}
	return -1
}

func NewCtxErrOnly() counter.Provider {
	return ctxErrOnly{}
}
