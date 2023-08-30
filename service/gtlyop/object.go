package gtlyop

import (
	"log"

	"github.com/viant/gtly"
	"github.com/viant/mly/service/config"
)

func NewObjectProvider(config *config.Model) (*gtly.Provider, error) {
	verbose := config.Debug
	inputs := config.Inputs

	var fields = make([]*gtly.Field, len(inputs))
	for i, field := range inputs {
		if verbose {
			log.Printf("[%s NewObjectProvider] %s %v", config.ID, field.Name, field.RawType())
		}

		fields[i] = &gtly.Field{
			Name: field.Name,
			Type: field.RawType(),
		}
	}
	provider, err := gtly.NewProvider("input", fields...)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
