package stat

import (
	"fmt"

	"github.com/viant/gmetric/counter"
	"github.com/viant/mly/shared/stat"
)

const (
	ReadErrorKey      = "readError"
	UnmarshalErrorKey = "unmarshalError"
)

type http struct{}

type ReadError struct{ Error error }

// implements fmt.Stringer
func (r ReadError) String() string { return r.Error.Error() }

// implements github.com/viant/gmetric/counter.CustomCounter
func (r ReadError) Aggregate(interface{}) {}

type UnmarshalError struct{ Error error }

// implements fmt.Stringer
func (r UnmarshalError) String() string { return r.Error.Error() }

// implements github.com/viant/gmetric/counter.CustomCounter
func (r UnmarshalError) Aggregate(interface{}) {}

func (p http) Keys() []string {
	return []string{
		ReadErrorKey,
		UnmarshalErrorKey,
	}
}

func (p http) Map(value interface{}) int {
	if value == nil {
		return -1
	}

	if v, ok := value.(string); ok {
		switch v {
		case ReadErrorKey:
			return 0
		case UnmarshalErrorKey:
			return 1
		}

		return -1
	}

	if _, ok := value.(ReadError); ok {
		return 0
	}

	if _, ok := value.(UnmarshalError); ok {
		return 1
	}

	fmt.Printf("%+v\n", value)

	return -1
}

// implements github.com/viant/gmetric/counter.CustomProvider
func (h http) NewCounter() counter.CustomCounter {
	return stat.NewTopK(5, 0)
}

func NewHttp() counter.Provider {
	return http{}
}
