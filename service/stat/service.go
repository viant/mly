package stat

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

const (
	Evaluate = "eval"
	Pending  = "pending"
	Invalid  = "invalid"
)

type provider struct{}

//Keys returns metric keys
func (p provider) Keys() []string {
	return []string{
		stat.ErrorKey,
		Evaluate,
		Pending,
		stat.Timeout,
		Invalid,
	}
}

//Map maps metric key into value index
func (p provider) Map(value interface{}) int {
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
			return 3
		case Invalid:
			return 4
		}
	}
	return -1
}

func NewProvider() counter.Provider {
	return &provider{}
}
