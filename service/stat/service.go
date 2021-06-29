package stat

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

const (
	EvalKey = "eval"
)

type service struct{}

func (p service) Keys() []string {
	return []string{
		stat.ErrorKey,
		EvalKey,
		stat.Timeout,
	}
}

func (p service) Map(value interface{}) int {
	if value == nil {
		return -1
	}
	switch val := value.(type) {
	case error:
		return 0
	case string:
		switch val {
		case EvalKey:
			return 1
		case stat.Timeout:
			return 2
		}
	}
	return -1
}

func NewService() counter.Provider {
	return &service{}
}
