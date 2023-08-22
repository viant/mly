package stat

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

type tfs struct{}

const Running = "running"

// implements github.com/viant/gmetric/counter.Provider
func (p tfs) Keys() []string {
	return []string{
		stat.ErrorKey,
		Running,
	}
}

// implements github.com/viant/gmetric/counter.Provider
func (p tfs) Map(value interface{}) int {
	if value == nil {
		return -1
	}

	switch val := value.(type) {
	case error:
		return 0
	case string:
		switch val {
		case Running:
			return 1
		}
	case stat.Dir:
		return 1
	}

	return -1
}

func (p tfs) NewCounter() counter.CustomCounter {
	return new(stat.Occupancy)
}

func NewTfs() counter.Provider {
	return &tfs{}
}
