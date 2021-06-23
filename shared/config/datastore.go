package config

import (
	"fmt"
	"github.com/viant/mly/common"
	"github.com/viant/mly/common/storable"
	"github.com/viant/mly/shared/config/datastore"
	"github.com/viant/scache"
	"reflect"
)

type Datastore struct {
	ID    string
	Cache *scache.Config
	*datastore.Reference
	L2            *datastore.Reference
	Storable      string
	Fields        []*Field
	storableTypes []reflect.Type
}

func (d *Datastore) StorableTypes() []reflect.Type {
	return d.storableTypes
}

func (d *Datastore) Init() {

}

func (d *Datastore) FieldsDescriptor(fields []*Field) error {
	d.Fields = fields
	d.storableTypes = make([]reflect.Type, 0)
	for _, field := range d.Fields {
		if err := field.Init(); err != nil {
			return err
		}
	}
	return nil
}

func (d *Datastore) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("datastore ID was empty")
	}
	if d.Reference != nil {
		if d.Dataset == "" {
			return fmt.Errorf("datastore Dataset was empty")
		}
		if d.Namespace == "" {
			return fmt.Errorf("datastore Namespace was empty")
		}
	}
	if d.Storable != "" {
		if _, err := storable.Singleton().Lookup(d.Storable);err != nil {
			return fmt.Errorf("unknown storable: %v, on datastore: %v", d.Storable, d.ID)
		}
		return fmt.Errorf("datastore ID was empty")
	}
	return nil
}

//Field represents a  default storable field descriptor
type Field struct {
	Name     string
	DataType string
	dataType reflect.Type
}

func (f *Field) Type() reflect.Type {
	return f.dataType
}

func (f *Field) Init() (err error) {
	if f.dataType != nil {
		return nil
	}
	f.dataType, err = common.DataType(f.DataType)
	return err
}

//NewFields create new fields
func NewFields(name string, dataType string) []*Field {
	field := &Field{Name: name, DataType: dataType}
	return []*Field{
		field,
	}
}
