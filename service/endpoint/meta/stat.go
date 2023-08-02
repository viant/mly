package meta

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

const (
	NotFound   = "http404"
	Dictionary = "dict"
	Config     = "cfg"
)

func NewProvider() counter.Provider {
	return stat.ErrorOnly()
}

type svp struct{}

func (p *svp) Keys() []string {
	return []string{
		stat.ErrorKey,
		NotFound,
		Dictionary,
		Config,
	}
}

func (v *svp) Map(value interface{}) int {
	if value == nil {
		return -1
	}
	switch val := value.(type) {
	case error:
		return 0
	case string:
		switch val {
		case NotFound:
			return 1
		case Dictionary:
			return 2
		case Config:
			return 3
		}
	}
	return -1
}

func NewServiceVP() counter.Provider {
	return &svp{}
}
