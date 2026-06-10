package router

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/platform"
	"github.com/viant/mly/service/triton"
	"github.com/viant/mly/shared"
	sharedrouter "github.com/viant/mly/shared/config/router"
)

type mockUnloader struct {
	tritonServer *mockTritonServer
	unloadCh     chan string
}

func (m *mockUnloader) ModelUnload(ctx context.Context, tritonModelName string) error {
	if m.tritonServer != nil {
		m.tritonServer.mu.Lock()
		defer m.tritonServer.mu.Unlock()

		if m.tritonServer.readyState == nil {
			m.tritonServer.readyState = make(map[string]bool)
		}

		m.tritonServer.readyState[tritonModelName] = false
	}

	ch := m.unloadCh

	if ch != nil {
		ch <- tritonModelName
	}

	return nil
}

type wrappedUnloader struct {
	tritonService *triton.Service

	wg *sync.WaitGroup
}

func (w *wrappedUnloader) UnloadModel(ctx context.Context, mlyModelID string, tritonModelName string) error {
	defer w.wg.Done()
	return w.tritonService.UnloadModel(ctx, mlyModelID, tritonModelName)
}

type mockTritonServer struct {
	mu sync.Mutex

	readyState   map[string]bool
	modelLoadErr map[string]error
}

func (m *mockTritonServer) ModelLoad(ctx context.Context, modelName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.modelLoadErr[modelName]; err != nil {
		return err
	}

	if m.readyState == nil {
		m.readyState = make(map[string]bool)
	}

	m.readyState[modelName] = true

	return nil
}

func TestRouter_applyRouterConfig_LoadsAndSwaps(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockUnloader{
		unloadCh: make(chan string, 2),
	}

	oldConfig := &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelA"},
			{EntityID: 2, ModelName: "modelB"},
		},
		GlobalModelName: "global-old",
	}

	makeSig := func() *domain.Signature {
		return &domain.Signature{
			Inputs: []domain.Input{
				{Name: "text", Index: 0, Type: reflect.TypeOf("")},
			},
			Outputs: []domain.Output{
				{Name: "score", Index: 0, DataType: "float32"},
			},
		}
	}

	router := &Router{
		unloader:      &triton.Service{Unloader: mockClient, Repository: triton.NewRepository()},
		routingConfig: oldConfig,
		routingMap: map[int]string{
			1: "modelA",
			2: "modelB",
		},
		routingTable: map[string]platform.PlatformEvaluator{
			"modelA": &mockEvaluator{signature: makeSig},
			"modelB": &mockEvaluator{signature: makeSig},
		},
		globalModel: &mockEvaluator{},
		debug:       true,
		makeRoutedEvaluator: func(modelName string) (platform.PlatformEvaluator, error) {
			return &mockEvaluator{signature: makeSig}, nil
		},
		unloadGauge: routerModelUnloadGauge.WithLabelValues("test_router"),
	}

	reusedModelB := router.routingTable["modelB"]

	newConfig := &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelB"},
			{EntityID: 3, ModelName: "modelC"},
		},
		GlobalModelName: "global-new",
	}

	if err := router.applyRouterConfig(ctx, newConfig); err != nil {
		t.Fatalf("applyRouterConfig returned error: %v", err)
	}

	waitForCalls(t, mockClient.unloadCh, 2)

	if router.routingConfig != newConfig {
		t.Fatalf("routerConfig pointer not updated")
	}

	expectedRouting := map[int]string{
		1: "modelB",
		3: "modelC",
	}

	if !reflect.DeepEqual(router.routingMap, expectedRouting) {
		t.Fatalf("routingMap mismatch, got %#v", router.routingMap)
	}

	if router.globalModel == nil {
		t.Fatalf("globalModel was not set")
	}

	if _, ok := router.routingTable["modelB"]; !ok {
		t.Fatalf("routingTable missing modelB")
	}

	if router.routingTable["modelB"] != reusedModelB {
		t.Fatalf("modelB evaluator was not reused")
	}

	if _, ok := router.routingTable["modelC"]; !ok {
		t.Fatalf("routingTable missing modelC")
	}

	if _, ok := router.routingTable["modelA"]; ok {
		t.Fatalf("routingTable still contains modelA")
	}
}

func TestRouter_applyRouterConfig_LoadError(t *testing.T) {
	ctx := context.Background()

	loadErr := errors.New("load failure")

	tritonServer := new(mockTritonServer)
	tritonServer.modelLoadErr = map[string]error{
		"modelX": loadErr,
	}

	mockClient := &mockUnloader{}

	oldConfig := &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelA"},
		},
	}

	signature := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "text", Index: 0, Type: reflect.TypeOf("")},
		},
		Outputs: []domain.Output{
			{Name: "score", Index: 0, DataType: "float32"},
		},
	}

	router := &Router{
		debug:      true,
		routerName: "load_error",

		unloader:      &triton.Service{Unloader: mockClient},
		routingConfig: oldConfig,
		routingMap: map[int]string{
			1: "modelA",
		},

		makeRoutedEvaluator: func(modelName string) (platform.PlatformEvaluator, error) {
			return &mockEvaluator{
				tritonServer: tritonServer,
				modelName:    modelName,
				signature:    func() *domain.Signature { return signature },
			}, nil
		},
		unloadGauge: routerModelUnloadGauge.WithLabelValues("load_error"),
	}

	newConfig := &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 2, ModelName: "modelX"},
		},
	}

	err := router.applyRouterConfig(ctx, newConfig)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}

	if !strings.Contains(err.Error(), "modelX") {
		t.Fatalf("expected error mentioning modelX, got %v", err)
	}

	if router.routingConfig != oldConfig {
		t.Fatalf("routerConfig should remain unchanged on error")
	}

	if !reflect.DeepEqual(router.routingMap, map[int]string{1: "modelA"}) {
		t.Fatalf("routingMap should remain unchanged on error")
	}

	if router.routingTable != nil {
		t.Fatalf("routingTable should not be replaced on error")
	}
}

func TestRouter_applyRouterConfig_signature(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockUnloader{
		unloadCh:     make(chan string, 1),
		tritonServer: &mockTritonServer{},
	}

	makeSig := func() *domain.Signature {
		return &domain.Signature{
			Inputs: []domain.Input{
				{Name: "text", Index: 0, Type: reflect.TypeOf("")},
			},
			Outputs: []domain.Output{
				{Name: "score", Index: 0, DataType: "float32"},
			},
		}
	}

	cfg := &config.Model{
		ID:       "test_signature",
		Debug:    true,
		Mode:     "router",
		Platform: "triton",
		Router: &config.RouterConfig{
			ConfigURL: "memory://router-config",
			InputName: "router_id",
			Global: config.GlobalModelConfig{
				Exists: true,
			},
			Output: config.OutputConfig{
				FieldName: "model_id",
			},
		},
		Triton: &config.TritonConfig{
			ServerID: "test_server",
		},
	}

	cfg.Init(nil)

	var router *Router
	var err error

	router, err = newRouter(cfg, nil, map[string]UnloadService{
		"test_server": &triton.Service{Unloader: mockClient},
	}, func(modelName string) (platform.PlatformEvaluator, error) {
		return &mockEvaluator{signature: makeSig}, nil
	})

	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}

	newConfig := &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "model1"},
			{EntityID: 2, ModelName: "model2"},
		},
		GlobalModelName: "global-model",
	}

	if err := router.applyRouterConfig(ctx, newConfig); err != nil {
		t.Fatalf("applyRouterConfig returned error: %v", err)
	}

	detectedInputs := map[string]struct{}{}
	for _, input := range router.ioState.signature.Inputs {
		detectedInputs[input.Name] = struct{}{}
	}

	expectedInputs := map[string]struct{}{
		"text":      {},
		"router_id": {},
	}

	assert.Equal(t, expectedInputs, detectedInputs)
	assert.Equal(t, 1, router.ioState.routerInputOffset)

	params := []interface{}{
		[][]string{{"a"}, {"abcd"}}, // text
		[][]int64{{1}, {2}},         // router_id
	}

	results, err := router.Predict(ctx, params)
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}

	assert.Equal(t, 2, len(results))
}

func TestRouter_applyRouterConfig_multiOutputSignature(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockUnloader{
		tritonServer: &mockTritonServer{},
	}

	makeSig := func() *domain.Signature {
		return &domain.Signature{
			Inputs: []domain.Input{
				{Name: "text", Index: 0, Type: reflect.TypeOf("")},
			},
			Outputs: []domain.Output{
				{Name: "score", Index: 0, DataType: "float32"},
				{Name: "calibration", Index: 1, DataType: "float32"},
			},
		}
	}

	cfg := &config.Model{
		ID:       "test_multi_output_signature",
		Debug:    true,
		Mode:     "router",
		Platform: "triton",
		Router: &config.RouterConfig{
			ConfigURL: "memory://router-config",
			InputName: "router_id",
			Global: config.GlobalModelConfig{
				PredictionReplacements: []config.PredictionReplacement{
					{Name: "score", Type: "float32", Value: 0.0},
					{Name: "calibration", Type: "float32", Value: 0.0},
				},
			},
			Output: config.OutputConfig{
				FieldName: "model_id",
			},
		},
		Triton: &config.TritonConfig{
			ServerID: "test_server",
		},
	}

	cfg.Init(nil)

	router, err := newRouter(cfg, nil, map[string]UnloadService{
		"test_server": &triton.Service{Unloader: mockClient},
	}, func(modelName string) (platform.PlatformEvaluator, error) {
		return &mockEvaluator{signature: makeSig}, nil
	})
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}

	newConfig := &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "model1"},
			{EntityID: 2, ModelName: "model2"},
		},
	}

	if err := router.applyRouterConfig(ctx, newConfig); err != nil {
		t.Fatalf("applyRouterConfig returned error: %v", err)
	}

	outputNames := make([]string, 0, len(router.ioState.signature.Outputs))
	for _, output := range router.ioState.signature.Outputs {
		outputNames = append(outputNames, output.Name)
	}

	assert.Equal(t, []string{"score", "calibration", "model_id"}, outputNames)
	assert.Equal(t, 1, router.ioState.routerInputOffset)
}

func TestRouter_applyRouterConfig_configuredOutputOrder(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockUnloader{
		tritonServer: &mockTritonServer{},
	}

	model1Sig := func() *domain.Signature {
		return &domain.Signature{
			Inputs: []domain.Input{
				{Name: "text", Index: 0, Type: reflect.TypeOf("")},
			},
			Outputs: []domain.Output{
				{Name: "score", Index: 0, DataType: "float32"},
				{Name: "calibration", Index: 1, DataType: "float32"},
			},
		}
	}

	model2Sig := func() *domain.Signature {
		return &domain.Signature{
			Inputs: []domain.Input{
				{Name: "text", Index: 0, Type: reflect.TypeOf("")},
			},
			Outputs: []domain.Output{
				{Name: "calibration", Index: 0, DataType: "float32"},
				{Name: "score", Index: 1, DataType: "float32"},
			},
		}
	}

	predictValues := func(valuesByName map[string]float32) func([]interface{}, *domain.Signature) ([]interface{}, error) {
		return func(params []interface{}, signature *domain.Signature) ([]interface{}, error) {
			batchSize := len(params[0].([][]string))
			outputs := make([]interface{}, len(signature.Outputs))
			for i, output := range signature.Outputs {
				value := valuesByName[output.Name]
				batch := make([][]float32, batchSize)
				for j := range batch {
					batch[j] = []float32{value}
				}
				outputs[i] = batch
			}
			return outputs, nil
		}
	}

	cfg := &config.Model{
		ID:       "test_configured_output_order",
		Debug:    true,
		Mode:     "router",
		Platform: "triton",
		Router: &config.RouterConfig{
			ConfigURL: "memory://router-config",
			InputName: "router_id",
			Global: config.GlobalModelConfig{
				PredictionReplacements: []config.PredictionReplacement{
					{Name: "score", Type: "float32", Value: 0.0},
					{Name: "calibration", Type: "float32", Value: 0.0},
				},
			},
			Output: config.OutputConfig{
				FieldName: "model_id",
			},
		},
		MetaInput: shared.MetaInput{
			Outputs: []*shared.Field{
				{Name: "calibration", DataType: "float32"},
				{Name: "model_id", DataType: "string"},
				{Name: "score", DataType: "float32"},
			},
		},
		Triton: &config.TritonConfig{
			ServerID: "test_server",
		},
	}

	cfg.Init(nil)

	router, err := newRouter(cfg, nil, map[string]UnloadService{
		"test_server": &triton.Service{Unloader: mockClient},
	}, func(modelName string) (platform.PlatformEvaluator, error) {
		switch modelName {
		case "model1":
			return &mockEvaluator{
				modelName:    modelName,
				signature:    model1Sig,
				predictor:    predictValues(map[string]float32{"score": 1, "calibration": 2}),
				tritonServer: mockClient.tritonServer,
			}, nil
		case "model2":
			return &mockEvaluator{
				modelName:    modelName,
				signature:    model2Sig,
				predictor:    predictValues(map[string]float32{"score": 10, "calibration": 20}),
				tritonServer: mockClient.tritonServer,
			}, nil
		default:
			return nil, errors.New("unexpected model")
		}
	})
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}

	if err := router.applyRouterConfig(ctx, &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "model1"},
			{EntityID: 2, ModelName: "model2"},
		},
	}); err != nil {
		t.Fatalf("applyRouterConfig returned error: %v", err)
	}

	outputNames := make([]string, 0, len(router.ioState.signature.Outputs))
	for _, output := range router.ioState.signature.Outputs {
		outputNames = append(outputNames, output.Name)
	}
	assert.Equal(t, []string{"calibration", "model_id", "score"}, outputNames)

	results, err := router.Predict(ctx, []interface{}{
		[][]string{{"a"}, {"b"}},
		[][]int64{{1}, {2}},
	})
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}

	assert.Equal(t, [][]float32{{2}, {20}}, results[0])
	assert.Equal(t, [][]string{{"model1"}, {"model2"}}, results[1])
	assert.Equal(t, [][]float32{{1}, {10}}, results[2])
}

func TestRouter_applyRouterConfig_configuredOutputValidation(t *testing.T) {
	tests := []struct {
		name              string
		configuredOutputs []*shared.Field
		expectedErr       string
	}{
		{
			name: "missing configured output",
			configuredOutputs: []*shared.Field{
				{Name: "missing", DataType: "float32"},
			},
			expectedErr: "configured output missing was not found in model outputs",
		},
		{
			name: "unconfigured downstream output",
			configuredOutputs: []*shared.Field{
				{Name: "score", DataType: "float32"},
			},
			expectedErr: "model outputs not present in configured outputs: calibration",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mockClient := &mockUnloader{
				tritonServer: &mockTritonServer{},
			}

			cfg := &config.Model{
				ID:       "test_configured_output_validation",
				Debug:    true,
				Mode:     "router",
				Platform: "triton",
				Router: &config.RouterConfig{
					ConfigURL: "memory://router-config",
					InputName: "router_id",
					Global: config.GlobalModelConfig{
						PredictionReplacements: []config.PredictionReplacement{
							{Name: "score", Type: "float32", Value: 0.0},
							{Name: "calibration", Type: "float32", Value: 0.0},
						},
					},
				},
				MetaInput: shared.MetaInput{
					Outputs: test.configuredOutputs,
				},
				Triton: &config.TritonConfig{
					ServerID: "test_server",
				},
			}

			cfg.Init(nil)

			router, err := newRouter(cfg, nil, map[string]UnloadService{
				"test_server": &triton.Service{Unloader: mockClient},
			}, func(modelName string) (platform.PlatformEvaluator, error) {
				return &mockEvaluator{
					modelName:    modelName,
					tritonServer: mockClient.tritonServer,
					signature: func() *domain.Signature {
						return &domain.Signature{
							Inputs: []domain.Input{
								{Name: "text", Index: 0, Type: reflect.TypeOf("")},
							},
							Outputs: []domain.Output{
								{Name: "score", Index: 0, DataType: "float32"},
								{Name: "calibration", Index: 1, DataType: "float32"},
							},
						}
					},
				}, nil
			})
			if err != nil {
				t.Fatalf("NewRouter error: %v", err)
			}

			err = router.applyRouterConfig(ctx, &sharedrouter.RoutingConfig{
				EntityMapping: []sharedrouter.EntityKV{
					{EntityID: 1, ModelName: "model1"},
				},
			})
			if err == nil {
				t.Fatalf("expected error but got nil")
			}
			assert.Contains(t, err.Error(), test.expectedErr)
		})
	}
}

func TestRouter_applyRouterConfig_sharedTritonServer(t *testing.T) {
	tritonServer := &mockTritonServer{}
	modelUnloader := &mockUnloader{
		tritonServer: tritonServer,
	}

	repository := triton.NewRepository()
	tritonService := &triton.Service{
		Unloader:   modelUnloader,
		Repository: repository,
	}

	wrappedService := &wrappedUnloader{
		tritonService: tritonService,
		wg:            &sync.WaitGroup{},
	}

	unloaders := map[string]UnloadService{
		"test_server": wrappedService,
	}

	cfgA := &config.Model{
		ID:       "test_shared_a",
		Debug:    true,
		Mode:     "router",
		Platform: "triton",
		Router: &config.RouterConfig{
			ConfigURL: "memory://router-config",
			Global: config.GlobalModelConfig{
				PredictionReplacements: []config.PredictionReplacement{
					{
						Name:  "score",
						Type:  "float32",
						Value: 0.0,
					},
				},
			},
		},
		Triton: &config.TritonConfig{
			ServerID: "test_server",
		},
	}

	cfgB := &config.Model{
		ID:       "test_shared_b",
		Debug:    true,
		Mode:     "router",
		Platform: "triton",
		Router: &config.RouterConfig{
			ConfigURL: "memory://router-config",
			Global: config.GlobalModelConfig{
				PredictionReplacements: []config.PredictionReplacement{
					{
						Name:  "score",
						Type:  "float32",
						Value: 0.0,
					},
				},
			},
		},
		Triton: &config.TritonConfig{
			ServerID: "test_server",
		},
	}

	cfgA.Init(nil)
	cfgB.Init(nil)

	newSig := func() *domain.Signature {
		return &domain.Signature{
			Inputs: []domain.Input{
				{Name: "text", Index: 0, Type: reflect.TypeOf("")},
			},
			Outputs: []domain.Output{
				{Name: "score", Index: 0, DataType: "float32"},
			},
		}
	}

	routerA, err := newRouter(cfgA, nil, unloaders, func(modelName string) (platform.PlatformEvaluator, error) {
		tritonService.RegisterUsage(cfgA.ID, modelName)
		return &mockEvaluator{modelName: modelName, signature: newSig, tritonServer: tritonServer}, nil
	})

	if err != nil {
		t.Fatalf("newRouter returned error: %v", err)
	}

	routerB, err := newRouter(cfgB, nil, unloaders, func(modelName string) (platform.PlatformEvaluator, error) {
		tritonService.RegisterUsage(cfgB.ID, modelName)
		return &mockEvaluator{modelName: modelName, signature: newSig, tritonServer: tritonServer}, nil
	})

	if err != nil {
		t.Fatalf("newRouter returned error: %v", err)
	}

	ctx := context.Background()

	// establish mappings for A and B using the same models

	if err = routerA.applyRouterConfig(ctx, &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelA"},
			{EntityID: 2, ModelName: "modelC"},
		},
	}); err != nil {
		t.Fatalf("applyRouterConfig A initial returned error: %v", err)
	}

	if err = routerB.applyRouterConfig(ctx, &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelA"},
		},
	}); err != nil {
		t.Fatalf("applyRouterConfig B inital returned error: %v", err)
	}

	// we expect model A and model C to be attempted to be unloaded
	wrappedService.wg.Add(2)

	// routerA will now get a different mapping
	if err = routerA.applyRouterConfig(ctx, &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelB"},
		},
	}); err != nil {
		t.Fatalf("applyRouterConfig A reload returned error: %v", err)
	}

	wrappedService.wg.Wait()

	assert.True(t, tritonServer.readyState["modelA"], "modelA should still be loaded")
	assert.False(t, tritonServer.readyState["modelC"], "modelC should be unloaded")

	// we expect model A to be attempted to be unloaded
	wrappedService.wg.Add(1)

	if err = routerB.applyRouterConfig(ctx, &sharedrouter.RoutingConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelB"},
			{EntityID: 2, ModelName: "modelC"},
		},
	}); err != nil {
		t.Fatalf("applyRouterConfig B reload returned error: %v", err)
	}

	wrappedService.wg.Wait()

	assert.False(t, tritonServer.readyState["modelA"], "modelA should be unloaded")
	assert.True(t, tritonServer.readyState["modelB"], "modelB should still be loaded")
	assert.True(t, tritonServer.readyState["modelC"], "modelC should be loaded")
}

func waitForCalls(t *testing.T, ch <-chan string, count int) []string {
	t.Helper()
	var out []string
	for i := 0; i < count; i++ {
		select {
		case v := <-ch:
			out = append(out, v)
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for call %d/%d", i+1, count)
		}
	}
	return out
}
