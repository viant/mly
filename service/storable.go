package service

import (
	"github.com/viant/mly/common"
	"github.com/viant/mly/common/storable"
	"github.com/viant/mly/shared/config"
)

func getStorable(cfg *config.Datastore) func() common.Storable {
	result, err := storable.Singleton().Lookup(cfg.Storable)
	if err == nil && result != nil {
		return result
	} //otherwise return default storable
	return func() common.Storable {
		return storable.New(cfg.Fields)
	}
}
