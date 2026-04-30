package config

import "fmt"

type RouterConfig struct {
	// Required if Model.Mode is "router".
	ConfigURL string

	// Required
	InputName string `json:",omitempty" yaml:",omitempty"`

	// Unimplemented.
	// If true, the router will batch the requests to the backend.
	BatchBackend bool `json:",omitempty" yaml:",omitempty"`

	// The maximum number of concurrent requests to the backend.
	// Defaults to 50.
	Workers int `json:",omitempty" yaml:",omitempty"`

	// The maximum number of requests to queue.
	// Defaults to 1000.
	MaxQueueSize int `json:",omitempty" yaml:",omitempty"`

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
	// The Output must exist in Model.MetaInput.Outputs.
	FieldName string `json:",omitempty" yaml:",omitempty"`

	// If the global model is used, this will be used as the global model name
	// If RouterConfig.Global.Exists is false, this will be ignored.
	GlobalModelOverride string `json:",omitempty" yaml:",omitempty"`

	// If no model is used, this will be used as the model ID
	// If RouterConfig.Global.Exists is true, this will be ignored.
	// Defaults to "none".
	NoModelID string `json:",omitempty" yaml:",omitempty"`
}

func (o *RouterConfig) Init() {
	if o.Workers == 0 {
		o.Workers = 50
	}

	if o.MaxQueueSize == 0 {
		o.MaxQueueSize = 1000
	}
}

func (o *RouterConfig) Validate() error {
	if o.ConfigURL == "" {
		return fmt.Errorf("config URL is required")
	}

	if o.Workers == 0 {
		return fmt.Errorf("workers must be greater than 0")
	}

	if o.MaxQueueSize == 0 {
		return fmt.Errorf("max queue size must be greater than 0")
	}

	if o.InputName == "" {
		return fmt.Errorf("input name is required")
	}

	if !o.Global.Exists && len(o.Global.PredictionReplacements) == 0 {
		return fmt.Errorf("global model does not exist but no prediction replacements were provided")
	}

	if o.Output.NoModelID == "" {
		o.Output.NoModelID = "none"
	}

	return nil
}
