package server

import (
	"context"
	"github.com/viant/mly/service/endpoint"
)

//Config represents a config
type Config struct {
	endpoint.Config
}

//NewConfigFromURL creates a new config from URL
func NewConfigFromURL(ctx context.Context, URL string) (*Config, error) {
	cfg := &Config{}
	if err := cfg.LoadFromURL(ctx, URL, cfg); err != nil {
		return nil, err
	}
	cfg.Init()
	return cfg, nil
}
