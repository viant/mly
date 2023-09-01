package stat

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

const (
	Evaluate   = "eval"
	Invalid    = "invalid"
	Overloaded = "overloaded"
)

type service struct{}

// implements github.com/viant/gmetric/counter.Provider
func (e service) Keys() []string {
	return []string{
		stat.ErrorKey,
		Evaluate,
		Pending,
		// Deprecated
		stat.Timeout,
		Invalid,
		stat.Canceled,
		stat.DeadlineExceeded,
		Overloaded,
	}
}

// implements github.com/viant/gmetric/counter.Provider
func (e service) Map(value interface{}) int {
	if value == nil {
		return -1
	}
	switch val := value.(type) {
	case error:
		return 0
	case string:
		switch val {
		case Evaluate:
			return 1
		case Pending:
			return 2
		case stat.Timeout:
			// Deprecated
			return 3
		case Invalid:
			return 4
		case stat.Canceled:
			return 5
		case stat.DeadlineExceeded:
			return 6
		case Overloaded:
			return 7
		}
	}

	return -1
}

func NewProvider() counter.Provider {
	return &service{}
}
