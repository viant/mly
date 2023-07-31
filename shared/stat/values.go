package stat

import (
	"errors"

	"github.com/viant/gmetric/stat"
)

// Values is a utility to pass multiple events to gmetric.
type Values []interface{}

func (v *Values) Append(item interface{}) {
	var ri interface{}
	switch item.(type) {
	case error:
		// due to the way metrics are exported via Prometheus, we don't want to change the value
		ri = errors.New(stat.ErrorKey)
	default:
		ri = item
	}

	*v = append(*v, ri)
}

func (v *Values) Values() []interface{} {
	return *v
}

func NewValues() *Values {
	return &Values{}
}
