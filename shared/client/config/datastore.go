package config

import (
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/config"
	"github.com/viant/mly/shared/config/datastore"
)

//Remote represents client datastore
type Remote struct {
	Connections []*datastore.Connection
	config.Datastore
	shared.MetaInput
}

func (d *Remote) Init() {
	d.MetaInput.Init()
}

func (d *Remote) Validate() error {
	//if len(d.Connections) == 0 && d.Datastore.ID != "" {
	//	return fmt.Errorf("connection was empty")
	//}
	//var connections = map[string]bool{}
	//for _, c := range d.Connections {
	//	connections[c.ID] = true
	//}
	//if _, ok := connections[d.Datastore.ID]; !ok {
	//	return fmt.Errorf("unknown datastore connection: %v", d.Datastore.ID)
	//}
	return nil
}
