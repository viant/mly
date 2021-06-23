package stat

import (
	"github.com/viant/gmetric/counter"
)

type service struct{}

func (p service) Keys() []string {
	return []string{
		ErrorKey,
		Timeout,
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
		case Timeout:
			return 1
		}
	}
	return -1
}


func NewService() counter.Provider {
	return &service{}
}
