package stat

import "github.com/viant/gmetric/counter"

// shared error-only multi operation tracker provider
type errorOnly struct{}

func (p errorOnly) Keys() []string {
	return []string{
		ErrorKey,
	}
}

func (p errorOnly) Map(value interface{}) int {
	if value == nil {
		return -1
	}

	if _, ok := value.(error); ok {
		return 0
	}

	return -1
}

func ErrorOnly() counter.Provider {
	return &errorOnly{}
}
