package server

import (
	"context"
	"github.com/viant/mly/example/sls"
	"github.com/viant/mly/example/vec"
	"github.com/viant/mly/service/domain/transformer"
	"github.com/viant/mly/service/endpoint"
	common "github.com/viant/mly/shared/common"
	storable "github.com/viant/mly/shared/common/storable"
)

//RunApp run application
func RunApp(Version string, args []string) {
	ctx := context.Background()
	storable.Singleton().Register("sls", func() common.Storable {
		return &sls.Record{}
	})
	storable.Singleton().Register("vec", func() common.Storable {
		return &vec.Record{}
	})
	endpoint.RunAppWithConfig(Version, args, func(options *endpoint.Options) (*endpoint.Config, error) {
		config, err := NewConfigFromURL(ctx, options.ConfigURL)
		if err != nil {
			return nil, err
		}
		transformer.Register("sls", sls.Transform)
		transformer.Register("vec", vec.Transform)
		return &config.Config, err
	})
}
