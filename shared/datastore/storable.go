package datastore

import (
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
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
