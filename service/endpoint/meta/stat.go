package meta

import (
	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

type vp struct{}

const (
	NotFound   = "http404"
	Dictionary = "dict"
	Config     = "cfg"
)

func (p *vp) Keys() []string {
	return []string{
		stat.ErrorKey,
	}
}

func (v *vp) Map(value interface{}) int {
	if value == nil {
		return -1
	}

	if _, ok := value.(error); ok {
		return 0
	}

	return -1
}

func NewProvider() counter.Provider {
	return &vp{}
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
