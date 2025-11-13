package platform

import (
	"context"

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

	// Signature returns underlying model's signature
	Signature() *domain.Signature

	// Dictionary returns vocabulary if available
	Dictionary() *common.Dictionary

	// Inputs returns the model input definitions for request validation
	Inputs() map[string]*domain.Input

	// Stats returns platform-specific live metrics, for debugging
	Stats(stats map[string]interface{})

	// Close releases resources
	Close() error

	// ReloadIfNeeded will update models as needed, and check their health.
	// For in-process models (TensorFlow), this will check if the underlying models need to be updated.
	// For external models (Triton), this will check Triton models' health.
	ReloadIfNeeded(ctx context.Context) error
}
