package server

import (
	"context"

	"github.com/viant/mly/service/endpoint"
)

func RunApp(Version string, args []string) {
	ctx := context.Background()

	endpoint.RunAppWithConfig(Version, args, func(options *endpoint.Options) (*endpoint.Config, error) {
		config, err := NewConfigFromURL(ctx, options.ConfigURL)
		if err != nil {
			return nil, err
		}

		return &config.Config, err
	})
}
