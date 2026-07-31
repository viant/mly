package config

import "fmt"

type RouterConfig struct {
	// Required if Model.Mode is "router".
	ConfigURL string

	// Required name of the input that will route the request to the backend.
	InputName string `json:",omitempty" yaml:",omitempty"`

	// ForceBatchSize1 controls whether the router sends individual samples or batches by model.
	// When false (default), requests within a single Predict() call that route to the
	// same model evaluator are grouped into a single batched prediction call.
	// When true, each sample is sent as an individual prediction request with batch size 1.
	ForceBatchSize1 bool `json:",omitempty" yaml:",omitempty"`

	// The maximum number of concurrent batches dispatched to model evaluators.
	// Defaults to 50.
	Workers int `json:",omitempty" yaml:",omitempty"`

	// The maximum number of batches to queue before rejecting.
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

	if o.Output.NoModelID == "" {
		o.Output.NoModelID = "none"
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

	return nil
}
