package config

import (
	"github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/config/datastore"
)

type Datastore struct {
	Connections []*datastore.Connection
	config.Datastore
	KeyFields []string
}
