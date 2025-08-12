package platform

import (
	"context"
	"fmt"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/shared/common"
)

// ModelPlatform represents the different model platforms supported
type ModelPlatform string

const (
	PlatformTensorFlow ModelPlatform = "tensorflow"
	PlatformTriton     ModelPlatform = "triton"
)

// PlatformEvaluator defines the interface that all platform-specific evaluators must implement
type PlatformEvaluator interface {
	// Predict performs model inference with the given parameters
	Predict(ctx context.Context, params []interface{}) ([]interface{}, error)

	// Signature returns model signature information
	Signature() interface{}

	// Dictionary returns vocabulary dictionary if available
	Dictionary() *common.Dictionary

	// Inputs returns the model input definitions for request validation
	Inputs() map[string]interface{}

	// Stats returns platform-specific statistics
	Stats(stats map[string]interface{})

	// Close releases resources
	Close() error
}

// PlatformRouter routes inference requests to the appropriate platform evaluator
type PlatformRouter interface {
	// Predict routes the request to the appropriate platform evaluator
	Predict(ctx context.Context, params []interface{}) ([]interface{}, error)

	// Signature returns the signature from the active evaluator
	Signature() interface{}

	// Dictionary returns the dictionary from the active evaluator
	Dictionary() *common.Dictionary

	// Stats aggregates statistics from all evaluators
	Stats(stats map[string]interface{})

	// Close closes all evaluators
	Close() error
}

// Router implements PlatformRouter
type Router struct {
	config    *config.Model
	Evaluator PlatformEvaluator // Exported for access from service package
	platform  ModelPlatform
}

// NewRouter creates a new platform router with the specified evaluator
func NewRouter(config *config.Model, evaluator PlatformEvaluator, platform ModelPlatform) *Router {
	return &Router{
		config:    config,
		Evaluator: evaluator,
		platform:  platform,
	}
}

// Predict routes the request to the platform evaluator
func (r *Router) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	if r.Evaluator == nil {
		return nil, fmt.Errorf("no evaluator configured for model %s", r.config.ID)
	}

	return r.Evaluator.Predict(ctx, params)
}

// Signature returns the signature from the active evaluator
func (r *Router) Signature() interface{} {
	if r.Evaluator == nil {
		return nil
	}
	return r.Evaluator.Signature()
}

// Dictionary returns the dictionary from the active evaluator
func (r *Router) Dictionary() *common.Dictionary {
	if r.Evaluator == nil {
		return nil
	}
	return r.Evaluator.Dictionary()
}

// Stats aggregates statistics from the evaluator
func (r *Router) Stats(stats map[string]interface{}) {
	if r.Evaluator != nil {
		// Add platform identifier to stats
		stats["platform"] = string(r.platform)
		r.Evaluator.Stats(stats)
	}
}

// Close closes the evaluator
func (r *Router) Close() error {
	if r.Evaluator == nil {
		return nil
	}
	return r.Evaluator.Close()
}

// GetPlatform returns the platform type
func (r *Router) GetPlatform() ModelPlatform {
	return r.platform
}
