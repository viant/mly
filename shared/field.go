package shared

import (
	"reflect"
)

type (
	Field struct {
		Name     string
		Index    int
		DataType string `json:",omitempty" yaml:",omitempty"`

		Auxiliary bool `json:",omitempty" yaml:",omitempty"`
		Wildcard  bool `json:",omitempty" yaml:",omitempty"`
		Precision int  `json:",omitempty" yaml:",omitempty"`

		rawType reflect.Type
	}

	Fields []*Field

	MetaInput struct {
		Inputs    []*Field
		KeyFields []string `json:",omitempty" yaml:",omitempty"` // Deprecated: use Field.Wildcard
		Auxiliary []string `json:",omitempty" yaml:",omitempty"` // Deprecated: use Field.Auxiliary
		Outputs   []*Field `json:",omitempty" yaml:",omitempty"`
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

func (f *Field) SetRawType(t reflect.Type) {
	switch t.Kind() {
	case reflect.String:
		f.DataType = "string"
	case reflect.Float32:
		f.DataType = "float"
	case reflect.Int64:
		f.DataType = "int64"
	default:
		f.DataType = t.Kind().String()
	}
	f.rawType = t
}

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

func (m *MetaInput) Init() {
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
		if input.Auxiliary { //this is an input for post model prediction transformer
			m.Inputs[i].Index = -1 //unknown fields
		}
		if input.rawType == nil {
			input.rawType = reflect.TypeOf("")
		}
		m.Inputs[i].Index = i
	}
}
