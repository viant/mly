package config

import (
	"fmt"

	"github.com/viant/mly/service/tfmodel/batcher/config"
)

// ModelList represents model
type ModelList struct {
	Models []*Model
}

// Init initialises model list
func (l *ModelList) Init(bc *config.BatcherConfig) {
	if len(l.Models) == 0 {
		return
	}

	for _, model := range l.Models {
		model.Init(bc)
	}
}

// Validate validates model list
func (l *ModelList) Validate() error {
	if len(l.Models) == 0 {
		return fmt.Errorf("models were empty")
	}

	for _, model := range l.Models {
		if err := model.Validate(); err != nil {
			return fmt.Errorf("failed to validate model: %s, err: %w", model.ID, err)
		}
	}
	return nil
}
