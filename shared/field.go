package shared

import "reflect"

type (
	Field struct {
		Name      string
		Index     int
		Wildcard  bool   `json:",omitempty" yaml:",omitempty"`
		DataType  string `json:",omitempty" yaml:",omitempty"`
		rawType   reflect.Type
		Auxiliary bool `json:",omitempty" yaml:",omitempty"`
	}
	Fields []*Field

	MetaInput struct {
		Inputs    []*Field
		KeyFields []string `json:",omitempty" yaml:",omitempty"`
	}
)

func (d *MetaInput) KeysLen() int {
	count := 0
	for _, item := range d.Inputs {
		if item.Auxiliary {
			continue
		}
		count++
	}
	return count
}

func (f *Field) RawType() reflect.Type {
	return f.rawType
}

func (f *Field) SetRawType(t reflect.Type) {
	f.rawType = t
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
	}
	for i, input := range m.Inputs {
		if input.Auxiliary { //this is an input for post model prediction transformer
			m.Inputs[i].Index = -1 //uknonw fields
		}
		if input.rawType == nil {
			input.rawType = reflect.TypeOf("")
		}
		m.Inputs[i].Index = i
	}
}
