package platform

import (
	"fmt"
	"time"

	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/tfmodel"
	"golang.org/x/sync/semaphore"
)

// CreateEvaluator creates the appropriate platform evaluator based on the model configuration
func CreateEvaluator(cfg *config.Model, fs afs.Service, metrics *gmetric.Service, sema *semaphore.Weighted, maxEvaluatorWait time.Duration) (PlatformEvaluator, ModelPlatform, error) {
	platform := cfg.GetPlatform()

	switch platform {
	case "tensorflow":
		// Create TensorFlow service using existing code
		tfService := tfmodel.NewService(cfg, fs, metrics, sema, maxEvaluatorWait)
		evaluator := NewTensorFlowEvaluator(tfService)
		return evaluator, PlatformTensorFlow, nil

	case "triton":
		// Create Triton evaluator (stub for now)
		evaluator := NewTritonEvaluator(cfg)
		return evaluator, PlatformTriton, nil

	default:
		return nil, "", fmt.Errorf("unsupported platform: %s for model %s", platform, cfg.ID)
	}
}

// CreateRouter creates a platform router with the appropriate evaluator
func CreateRouter(cfg *config.Model, fs afs.Service, metrics *gmetric.Service, sema *semaphore.Weighted, maxEvaluatorWait time.Duration) (*Router, error) {
	evaluator, platform, err := CreateEvaluator(cfg, fs, metrics, sema, maxEvaluatorWait)
	if err != nil {
		return nil, err
	}

	return NewRouter(cfg, evaluator, platform), nil
}
