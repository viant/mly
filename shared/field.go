package shared

import (
	"fmt"
	"reflect"

	"github.com/viant/mly/shared/common"
)

type (
	Field struct {
		Name string

		// Position in cache key.
		Index int

		// The type of the field.
		// Supports "float" which maps to float32.
		// Otherwise, refer to reflect.Type.Name().
		DataType string `json:",omitempty" yaml:",omitempty"`

		// Indicates not an input for the model, but is eligible to be passed in a payload.
		Auxiliary bool `json:",omitempty" yaml:",omitempty"`

		// Indicates, on its own, that it is an input for the model, and that it should be directly input as a cache key.
		// Applies to string inputs.
		Wildcard bool `json:",omitempty" yaml:",omitempty"`

		// Indicates, on its own, that it is an input for the model,
		// and that it should be directly input as a cache key.
		Precision int `json:",omitempty" yaml:",omitempty"`

		// Used when unmarshaling and feeding into the model.
		rawType reflect.Type
	}

	Fields []*Field

	MetaInput struct {
		Inputs []*Field

		// This is used to order inputs and provide extra caching information to the client.
		// All inputs from the model will automatically be added here.
		KeyFields []string `json:",omitempty" yaml:",omitempty"`

		// Deprecated: use Field.Auxiliary
		// Any field specified in here will be added as a Field with auxiliary:true, datatype:string.
		// If the field exists by name, it will cause a panic.
		Auxiliary []string `json:",omitempty" yaml:",omitempty"`

		Outputs []*Field `json:",omitempty" yaml:",omitempty"`
	}
)

// implements sort.Interface.Len
func (f Fields) Len() int {
	return len(f)
}

// implements sort.Interface.Less
func (f Fields) Less(i, j int) bool {
	return f[i].Index < f[j].Index
}

// implements sort.Interface.Swap
func (f Fields) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f *Field) RawType() reflect.Type {
	return f.rawType
}

func (f *Field) DataTypeToRawType() {
	f.rawType = fieldDataTypeToRawType(f.DataType)
}

// fieldDataTypeToRawType is a subset of reverse Name() to reflect.Type
func fieldDataTypeToRawType(dataType string) reflect.Type {
	switch dataType {
	case "float":
		// provided as a convenience
		return reflect.TypeOf(float32(0))
	case "":
		// this case is treated as string in common.DataType(), but here it's not OK.
		panic(fmt.Sprintf("unsupported data type: %s", dataType))
	default:
		rawType, err := common.DataType(dataType)
		if err != nil {
			panic(fmt.Sprintf("unsupported data type: %s", dataType))
		}
		return rawType
	}
}

// SetRawType is used when pulling from the model.
func (f *Field) SetRawType(t reflect.Type) {
	f.DataType = t.Name()
	f.rawType = t
}

// TODO Deprecate
func (m *MetaInput) OutputIndex() map[string]int {
	var outputIndex = map[string]int{}
	if len(m.Outputs) == 0 {
		return outputIndex
	}
	for i, f := range m.Outputs {
		outputIndex[f.Name] = i
	}
	return outputIndex
}

func (m *MetaInput) OutputByName() map[string]*Field {
	var outputByName = map[string]*Field{}
	for _, f := range m.Outputs {
		outputByName[f.Name] = f
	}
	return outputByName
}

func (d *MetaInput) KeysLen() int {
	return len(d.Inputs)
}

func (m *MetaInput) FieldByName() map[string]*Field {
	var result = make(map[string]*Field)
	for i, f := range m.Inputs {
		result[f.Name] = m.Inputs[i]
	}
	return result
}

// Init operates slightly differently on the server versus the client.
// On the server, it is called after reading the configuration file.
// On the client, it is called after fetching the configuration from the server, which will have already processed it via reconcileIOFromSignature().
func (m *MetaInput) Init() {
	// TODO assess why this approach was taken - this condition could be improved by having a map to see if the field by name already exists
	if len(m.Inputs) == 0 {
		if len(m.KeyFields) > 0 {
			for _, field := range m.KeyFields {
				m.Inputs = append(m.Inputs, &Field{Name: field})
			}
		}
		if len(m.Auxiliary) > 0 {
			for _, field := range m.Auxiliary {
				m.Inputs = append(m.Inputs, &Field{Name: field, Auxiliary: true})
			}
		}
	}

	for i, input := range m.Inputs {
		if input.rawType == nil && input.DataType != "" {
			input.DataTypeToRawType()
		}

		m.Inputs[i].Index = i
	}
}
