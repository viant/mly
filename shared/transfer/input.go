package transfer

import (
	"github.com/viant/gtly"
)

type Input struct {
	BatchSize int
	Keys      Strings
	Values
	Unmapped Values // values that are not part of an input
}

func (i *Input) BatchMode() bool {
	return i.BatchSize > 0
}

func (i *Input) Init(size int) {
	i.Keys = Strings{}
	i.Values = make([]Value, size)
}

func (i *Input) KeyAt(index int) string {
	if len(i.Keys.Values) <= index {
		return ""
	}

	return i.Keys.Values[index]
}

func (i *Input) ObjectAt(provider *gtly.Provider, index int) *gtly.Object {
	result := provider.NewObject()
	for j := range i.Values {
		value := i.ValueAt(j)
		result.SetValue(value.Key(), value.ValueAt(index))
	}
	return result
}
