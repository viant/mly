package config

import (
	"fmt"

	"github.com/viant/mly/shared/common/storable"
	"github.com/viant/mly/shared/config/datastore"
	"github.com/viant/scache"
)

//Datastore represents datastore
type Datastore struct {
	ID                   string
	Cache                *scache.Config
	*datastore.Reference `json:",omitempty" yaml:",inline"`
	L2                   *datastore.Reference `json:",omitempty" yaml:",omitempty"`
	Storable             string
	Fields               []*storable.Field
	Disabled             bool `json:",omitempty" yaml:",omitempty"`
	Debug                bool `json:",omitempty" yaml:",omitempty"`
}

//Init initialises datastore
func (d *Datastore) Init() {
	if d.Reference == nil {
		d.Reference = &datastore.Reference{}
	}
	d.Reference.Init()
}

//FieldsDescriptor sets field descriptors
func (d *Datastore) FieldsDescriptor(fields []*storable.Field) error {
	d.Fields = fields
	for _, field := range d.Fields {
		if err := field.Init(); err != nil {
			return err
		}
	}
	return nil
}

//Validate checks if datastore settings are valid
func (d *Datastore) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("datastore ID was empty")
	}
	if d.Reference.Connection != "" {
		if d.Dataset == "" {
			return fmt.Errorf("datastore Dataset was empty")
		}
		if d.Namespace == "" {
			return fmt.Errorf("datastore Namespace was empty")
		}
	}
	if d.Storable != "" {
		if _, err := storable.Singleton().Lookup(d.Storable); err != nil {
			return fmt.Errorf("unknown storable: %v, on datastore: %v", d.Storable, d.ID)
		}
	}
	return nil
}
