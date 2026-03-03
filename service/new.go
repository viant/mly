package service

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/viant/afs"
	"github.com/viant/gmetric"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/platform/factory"
	"github.com/viant/mly/service/stat"
	"github.com/viant/mly/service/triton"
	"github.com/viant/mly/shared/datastore"
	sstat "github.com/viant/mly/shared/stat"
	"golang.org/x/sync/semaphore"
)

// NewArgs is an open-to-extension approach to keeping the NewV2() API invariant
// Potential tech debt: most likely we should be encapsulating parameters more appropriately.
// Natural encapsulation boundaries will emerge when we start seeing what New() also initializes along with just Service.
type NewArgs struct {
	Datastores       map[string]*datastore.Service
	TritonServices   map[string]*triton.Service
	Semaphore        *semaphore.Weighted
	MaxEvaluatorWait time.Duration

	HealthGauge *prometheus.GaugeVec
}

// New creates a service with platform router support
func New(
	ctx context.Context,
	cfg *config.Model,
	fs afs.Service,
	metrics *gmetric.Service,
	datastores map[string]*datastore.Service,
	tritonServices map[string]*triton.Service,
	sema *semaphore.Weighted,
	maxEvaluatorWait time.Duration,
	options ...Option,
) (*Service, error) {
	return NewV2(ctx, cfg, fs, metrics, NewArgs{
		Datastores:       datastores,
		TritonServices:   tritonServices,
		Semaphore:        sema,
		MaxEvaluatorWait: maxEvaluatorWait,
	}, options...)
}

// New creates a service with platform router support
func NewV2(
	ctx context.Context,
	cfg *config.Model,
	fs afs.Service,
	metrics *gmetric.Service,
	args NewArgs,
	options ...Option,
) (*Service, error) {

	if metrics == nil {
		metrics = gmetric.New()
	}

	location := reflect.TypeOf(Service{}).PkgPath()

	cfg.Init(nil)

	// Create platform evaluator context
	evaluatorContext, err := factory.CreateEvaluator(cfg, fs, metrics, args.Semaphore, args.MaxEvaluatorWait, args.TritonServices)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform evaluator for model %s: %w", cfg.ID, err)
	}

	srv := &Service{
		config:           cfg,
		evaluator:        evaluatorContext,
		useDatastore:     cfg.UseDictionary() && cfg.DataStore != "",
		serviceMetric:    metrics.MultiOperationCounter(location, cfg.ID+"Perf", cfg.ID+" service performance", time.Microsecond, time.Minute, 2, stat.NewProvider()),
		reloadPollTicker: time.NewTicker(time.Duration(cfg.ReloadPollIntervalSeconds) * time.Second),
		reloadTimeout:    time.Duration(cfg.ReloadTimeoutSeconds) * time.Second,
	}

	if args.HealthGauge != nil {
		srv.healthGauge = args.HealthGauge.With(prometheus.Labels{"model": cfg.ID})
	}

	// Set up reload metrics for platforms that support reloading
	srv.reloadMetric = metrics.MultiOperationCounter(location, cfg.ID+"Reload", cfg.ID+" reloading", time.Microsecond, time.Minute, 1, sstat.NewCtxErrOnly())

	for _, opt := range options {
		opt.Apply(srv)
	}

	err = srv.initializeService(ctx, cfg, fs, metrics, args.Datastores)
	if err != nil {
		return nil, err
	}

	go srv.pollModelReload()

	return srv, err
}
