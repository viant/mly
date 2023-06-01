package server

import (
	"context"

	"github.com/viant/mly/example/transformer/slf"
	slfmodel "github.com/viant/mly/example/transformer/slf/model"
	"github.com/viant/mly/service/domain/transformer"
	"github.com/viant/mly/service/endpoint"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
)

func RunApp(Version string, args []string) {
	ctx := context.Background()

	storableSrv := storable.Singleton()
	storableSrv.Register(slfmodel.Namespace, func() common.Storable {
		return new(slfmodel.Segmented)
	})

	endpoint.RunAppWithConfig(Version, args, func(options *endpoint.Options) (*endpoint.Config, error) {
		config, err := NewConfigFromURL(ctx, options.ConfigURL)
		if err != nil {
			return nil, err
		}

		transformer.Register(slfmodel.Namespace, slf.Transform)

		return &config.Config, err
	})
}
