package datastore

import (
	"github.com/viant/mly/common"
	"github.com/viant/mly/common/storable"
)

func getStorable(value interface{}) common.Storable {
	if value == nil {
		value = map[string]interface{}{}
	}
	aStorable, ok := value.(common.Storable)
	if !ok {
		aStorable = storable.NewGeneric(value)
	}
	return aStorable
}
