package platform

import (
	"context"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
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
	Signature() *domain.Signature

	// Dictionary returns vocabulary dictionary if available
	Dictionary() *common.Dictionary

	// Inputs returns the model input definitions for request validation
	Inputs() map[string]*domain.Input

	// Stats returns platform-specific statistics
	Stats(stats map[string]interface{})

	// Close releases resources
	Close() error

	// IsHealthy performs health check for the platform/model
	IsHealthy() bool

	// SetHealthStatus sets the health status pointer for centralized health reporting
	SetHealthStatus(healthPtr *int32)

	// SupportsHealthReporting returns true if this platform supports centralized health reporting
	SupportsHealthReporting() bool

	// ReloadIfNeeded performs model reload if needed (for platforms that support reload)
	ReloadIfNeeded(ctx context.Context) error

	// SupportsReload returns true if this platform supports model reloading
	SupportsReload() bool
}

// PlatformEvaluatorContext holds platform evaluator with context
type PlatformEvaluatorContext struct {
	Evaluator PlatformEvaluator
	Platform  ModelPlatform
	Config    *config.Model
}

// NewEvaluatorContext creates a new platform evaluator context
func NewEvaluatorContext(config *config.Model, evaluator PlatformEvaluator, platform ModelPlatform) *PlatformEvaluatorContext {
	return &PlatformEvaluatorContext{
		Evaluator: evaluator,
		Platform:  platform,
		Config:    config,
	}
}
