package stat

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

type eval struct{}

const Pending = "pending"

// implements github.com/viant/gmetric/counter.Provider
func (p eval) Keys() []string {
	return []string{
		stat.ErrorKey,
		stat.Timeout,
		Pending,
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
		}
	case stat.Dir:
		return 2
	}

	return -1
}

func (p eval) NewCounter() counter.CustomCounter {
	return new(stat.Occupancy)
}

func NewEval() counter.Provider {
	return &eval{}
}
