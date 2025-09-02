package platform

import (
	"context"
	"sync/atomic"

	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/shared/common"
)

// TensorFlowEvaluator wraps the existing TensorFlow service to implement PlatformEvaluator
type TensorFlowEvaluator struct {
	tfService *tfmodel.Service
}

// NewTensorFlowEvaluator creates a new TensorFlow evaluator wrapper
func NewTensorFlowEvaluator(tfService *tfmodel.Service) *TensorFlowEvaluator {
	return &TensorFlowEvaluator{
		tfService: tfService,
	}
}

// Predict delegates to the TensorFlow service
func (t *TensorFlowEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	return t.tfService.Predict(ctx, params)
}

// Signature delegates to the TensorFlow service
func (t *TensorFlowEvaluator) Signature() *domain.Signature {
	return t.tfService.Signature()
}

// Dictionary delegates to the TensorFlow service
func (t *TensorFlowEvaluator) Dictionary() *common.Dictionary {
	return t.tfService.Dictionary()
}

// Stats delegates to the TensorFlow service
func (t *TensorFlowEvaluator) Stats(stats map[string]interface{}) {
	t.tfService.Stats(stats)
}

// Close delegates to the TensorFlow service
func (t *TensorFlowEvaluator) Close() error {
	return t.tfService.Close()
}

// Inputs returns the model inputs for request validation
func (t *TensorFlowEvaluator) Inputs() map[string]*domain.Input {
	return t.tfService.Inputs()
}

// IsHealthy returns true if the TensorFlow model is healthy (reload successful)
func (t *TensorFlowEvaluator) IsHealthy() bool {
	if t.tfService.ReloadOK == nil {
		return false
	}
	return atomic.LoadInt32(t.tfService.ReloadOK) == 1
}

// SetHealthStatus sets the health status pointer for centralized health reporting
func (t *TensorFlowEvaluator) SetHealthStatus(healthPtr *int32) {
	t.tfService.ReloadOK = healthPtr
}

func (t *TensorFlowEvaluator) SupportsHealthReporting() bool {
	return true
}

// ReloadIfNeeded performs model reload if needed for TensorFlow models
func (t *TensorFlowEvaluator) ReloadIfNeeded(ctx context.Context) error {
	return t.tfService.ReloadIfNeeded(ctx)
}

// SupportsReload returns true since TensorFlow models support reloading
func (t *TensorFlowEvaluator) SupportsReload() bool {
	return true
}
