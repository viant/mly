package stat

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

type eval struct{}

const Pending = "pending"
const RLockEvaluator = "waitEvalLock"

// implements github.com/viant/gmetric/counter.Provider
func (p eval) Keys() []string {
	return []string{
		stat.ErrorKey,
		stat.Timeout,
		Pending,
		RLockEvaluator,
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
		}
	case stat.Dir:
		return 2
	case RLockDir:
		return 3
	}

	return -1
}

type RLockDir stat.Dir

// Does nothing but implement counter.CustomCounter
func (_ RLockDir) Aggregate(interface{}) {}

func (p eval) NewCounter() counter.CustomCounter {
	return new(stat.Occupancy)
}

func NewEval() counter.Provider {
	return &eval{}
}
