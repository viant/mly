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
	Predictor

	// Signature returns underlying model's signature.
	// This is expected to return non-nil after ReloadIfNeeded() succeeds.
	Signature() *domain.Signature

	// Dictionary returns vocabulary if available.
	Dictionary() *common.Dictionary

	// Inputs returns the model input definitions for request validation.
	// This will be invoked after at least 1 ReloadIfNeeded() succeeds.
	Inputs() map[string]*domain.Input

	// Stats returns platform-specific live metrics, for debugging
	Stats(stats map[string]interface{})

	Close() error

	// ReloadIfNeeded will update models as needed, check their health, and consolidate signatures, if implemented.
	// This can also be named EnsurePredictionPossible() or EnsureReady() or the like.
	// For in-process models (TensorFlow), this will check if the underlying models need to be updated.
	// For external models (Triton), this will use the Model Control API to load, unload, and check the health of Triton models.
	ReloadIfNeeded(ctx context.Context) error
}

type Predictor interface {
	// Predict performs model inference with the given parameters
	// params is expected to be [numInputs]([batchSize][1]T) (see service/request.Request.Feeds)
	// The return value should be [numOutputs]([batchSize][1]T), but may vary depending on the model.
	Predict(ctx context.Context, params []interface{}) ([]interface{}, error)
}
