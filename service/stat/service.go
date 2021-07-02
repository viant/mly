package stat

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

const (
	//Evaluate eval key
	Evaluate = "eval"
	Pending  = "pending"
)

type service struct{}

//Keys returns metric keys
func (p service) Keys() []string {
	return []string{
		stat.ErrorKey,
		Evaluate,
		Pending,
		stat.Timeout,
	}
}

//Map maps metric key into value index
func (p service) Map(value interface{}) int {
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
		}
	}
	return -1
}

//NewService creates service metrics
func NewService() counter.Provider {
	return &service{}
}
