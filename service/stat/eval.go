package stat

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

type eval struct{}

const (
	Pending        = "pending"
	RLockEvaluator = "waitEvalLock"
	NumElements    = "elements"
)

// implements github.com/viant/gmetric/counter.Provider
func (p eval) Keys() []string {
	return []string{
		stat.ErrorKey,
		stat.Timeout,
		Pending,
		RLockEvaluator,
		NumElements,
	}
}

// implements github.com/viant/gmetric/counter.Provider
func (p eval) Map(value interface{}) int {
	if value == nil {
		return -1
	}
	switch val := value.(type) {
	case error:
		return 0
	case string:
		switch val {
		case stat.Timeout:
			return 1
		case Pending:
			return 2
		case RLockEvaluator:
			return 3
		case NumElements:
			return 4
		}
	}
	return -1
}

func NewEval() counter.Provider {
	return &eval{}
}
