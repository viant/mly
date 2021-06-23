package config

import (
	"fmt"
	"github.com/viant/mly/shared/config/datastore"
)

type DatastoreList struct {
	Connections []*datastore.Connection
	Datastores  []*Datastore
}

func (d *DatastoreList) Init() {
	if len(d.Connections) == 0 || len(d.Datastores) == 0 {
		return
	}
	for i := range d.Connections {
		d.Connections[i].Init()
	}
	for i := range d.Datastores {
		d.Datastores[i].Init()
	}
}

func (d *DatastoreList) Validate() error {
	if len(d.Connections) == 0 && len(d.Datastores) == 0 {
		return nil
	}
	if len(d.Connections) > 0 && len(d.Datastores) == 0 {
		return fmt.Errorf("item were empty, but item defined")
	}
	if len(d.Connections) > 0 {
		for _, item := range d.Connections {
			if err := item.Validate(); err != nil {
				return err
			}
		}
	}
	for _, item := range d.Datastores {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}
