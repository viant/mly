package platform

import (
	"context"

	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/shared/common"
)

// TensorFlowEvaluator wraps the existing TensorFlow service to implement PlatformEvaluator
type TensorFlowEvaluator struct {
	TfService *tfmodel.Service // Exported for access from service package
}

// NewTensorFlowEvaluator creates a new TensorFlow evaluator wrapper
func NewTensorFlowEvaluator(tfService *tfmodel.Service) *TensorFlowEvaluator {
	return &TensorFlowEvaluator{
		TfService: tfService,
	}
}

// Predict delegates to the TensorFlow service
func (t *TensorFlowEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	return t.TfService.Predict(ctx, params)
}

// Signature delegates to the TensorFlow service
func (t *TensorFlowEvaluator) Signature() *domain.Signature {
	return t.TfService.Signature()
}

// Dictionary delegates to the TensorFlow service
func (t *TensorFlowEvaluator) Dictionary() *common.Dictionary {
	return t.TfService.Dictionary()
}

// Stats delegates to the TensorFlow service
func (t *TensorFlowEvaluator) Stats(stats map[string]interface{}) {
	t.TfService.Stats(stats)
}

// Close delegates to the TensorFlow service
func (t *TensorFlowEvaluator) Close() error {
	return t.TfService.Close()
}

// Inputs returns the model inputs for request validation
func (t *TensorFlowEvaluator) Inputs() map[string]interface{} {
	tfInputs := t.TfService.Inputs()
	// Convert to generic interface{} map for platform compatibility
	result := make(map[string]interface{})
	for k, v := range tfInputs {
		result[k] = v
	}
	return result
}

// SetReloadOK sets the reload status flag for TensorFlow models
func (t *TensorFlowEvaluator) SetReloadOK(reloadOK *int32) {
	t.TfService.ReloadOK = reloadOK
}

// ReloadIfNeeded performs model reload if needed for TensorFlow models
func (t *TensorFlowEvaluator) ReloadIfNeeded(ctx context.Context) error {
	return t.TfService.ReloadIfNeeded(ctx)
}

// SupportsReload returns true since TensorFlow models support reloading
func (t *TensorFlowEvaluator) SupportsReload() bool {
	return true
}
