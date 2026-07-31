package factory

import (
	"fmt"
	"time"

	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/platform"
	"github.com/viant/mly/service/platform/router"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/service/triton"
	"golang.org/x/sync/semaphore"
)

// CreateEvaluator creates the appropriate platform evaluator based on the model configuration
func CreateEvaluator(
	cfg *config.Model,

	fs afs.Service,
	metrics *gmetric.Service,
	sema *semaphore.Weighted,
	maxEvaluatorWait time.Duration,
	tritonServices map[string]*triton.Service,
) (platform.PlatformEvaluator, error) {
	p := cfg.GetPlatform()
	isRouter := cfg.IsRouter()

	switch p {
	case "tensorflow":
		// Create TensorFlow service using existing code
		tfService := tfmodel.NewService(cfg, fs, metrics, sema, maxEvaluatorWait)
		return tfService, nil

	case "triton":
		if isRouter {
			makeEvaluator := func(modelName string) (platform.PlatformEvaluator, error) {
				return triton.NewRoutedTritonEvaluator(modelName, cfg, tritonServices)
			}

			return router.NewRouter(cfg, fs, tritonServices, makeEvaluator)
		}

		return triton.NewTritonEvaluator(cfg, tritonServices)
	default:
		return nil, fmt.Errorf("unsupported platform: %s for model %s", p, cfg.ID)
	}
}
