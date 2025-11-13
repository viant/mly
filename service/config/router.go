package config

import "fmt"

type RouterConfig struct {
	// Required
	ConfigURL string

	// Required
	InputName string `json:",omitempty" yaml:",omitempty"`

	// Unimplemented.
	// If true, the router will batch the requests to the backend.
	BatchBackend bool `json:",omitempty" yaml:",omitempty"`

	Global GlobalModelConfig

	Output OutputConfig
}

type GlobalModelConfig struct {
	// Defaults to false
	Exists bool `json:",omitempty" yaml:",omitempty"`

	// Required if Exists is false
	PredictionReplacements []PredictionReplacement `json:",omitempty" yaml:",omitempty"`
}

type PredictionReplacement struct {
	Name  string
	Type  string
	Value any
}

type OutputConfig struct {
	// The name of the output field that contains the model ID.
	// If this is blank, then the model ID will not be part of the outputs.
	FieldName string `json:",omitempty" yaml:",omitempty"`

	// If the global model is used, this will be used as the global model name
	// If RouterConfig.Global.Exists is false, this will be ignored.
	GlobalModelOverride string `json:",omitempty" yaml:",omitempty"`

	// If no model is used, this will be used as the model ID
	// If RouterConfig.Global.Exists is true, this will be ignored.
	NoModelID string `json:",omitempty" yaml:",omitempty"`
}

func (o *RouterConfig) Validate() error {
	if o.ConfigURL == "" {
		return fmt.Errorf("config URL is required")
	}

	if o.InputName == "" {
		return fmt.Errorf("input name is required")
	}

	if !o.Global.Exists && len(o.Global.PredictionReplacements) == 0 {
		return fmt.Errorf("global model does not exist but no rediction replacements were provided")
	}

	return nil
}
