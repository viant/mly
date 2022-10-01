package service

import "github.com/viant/mly/shared/datastore"

type Option interface {
	Apply(c *Service)
}

type storerOption struct {
	storer datastore.Storer
}

//Apply metrics
func (o *storerOption) Apply(c *Service) {
	c.datastore = o.storer
	c.useDatastore = true
}

//WithDataStorer creates dictionary option
func WithDataStorer(storer datastore.Storer) Option {
	return &storerOption{storer: storer}
}
