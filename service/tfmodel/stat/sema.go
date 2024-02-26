package stat

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

type sema struct{}

const Waiting = "waiting"

// implements github.com/viant/gmetric/counter.Provider
func (p sema) Keys() []string {
	return []string{
		stat.ErrorKey,
		stat.Canceled,
		stat.DeadlineExceeded,
		Waiting,
	}
}

// implements github.com/viant/gmetric/counter.Provider
func (p sema) Map(value interface{}) int {
	if value == nil {
		return -1
	}

	switch val := value.(type) {
	case error:
		return 0
	case string:
		switch val {
		case stat.Canceled:
			return 1
		case stat.DeadlineExceeded:
			return 2
		case Waiting:
			return 3
		}
	case stat.Dir:
		return 3
	}

	return -1
}

func (p sema) NewCounter() counter.CustomCounter {
	return new(stat.Occupancy)
}

func NewSema() counter.Provider {
	return &sema{}
}
