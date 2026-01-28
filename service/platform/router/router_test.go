package router

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/viant/mly/shared/common"
	sharedrouter "github.com/viant/mly/shared/config/router"
)

// --- Router Predict scaffolds ---

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

type mockEvaluator struct {
	tritonServer *mockTritonServer

	modelName string
	signature func() *domain.Signature
}

func (m *mockEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	inputs := m.signature().Inputs
	if len(inputs) != len(params) {
		return nil, fmt.Errorf("mock error: expected %d inputs, got %d", len(inputs), len(params))
	}

	// params is expected to have a single non-router input in this test: [][]string with shape [1][1]
	var v string
	switch typed := params[0].(type) {
	case [][]string:
		v = typed[0][0]
	default:
		tval := reflect.TypeOf(params[0])
		panic("unexpected input type in mock Predict(): " + tval.String())
	}
	// simple function: length of string as float32
	out := [][]float32{{float32(len(v))}}
	return []interface{}{out}, nil
}

func (m *mockEvaluator) Signature() *domain.Signature     { return m.signature() }
func (m *mockEvaluator) Dictionary() *common.Dictionary   { return nil }
func (m *mockEvaluator) Inputs() map[string]*domain.Input { return nil }
func (m *mockEvaluator) Stats(map[string]interface{})     {}
func (m *mockEvaluator) Close() error                     { return nil }

func (m *mockEvaluator) ReloadIfNeeded(ctx context.Context) error {
	if m.tritonServer == nil {
		return nil
	}

	return m.tritonServer.ModelLoad(ctx, m.modelName)
}

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

func TestRouter_Predict(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		routerConfig *config.RouterConfig
		verifier     func(t *testing.T, results []interface{})
	}{
		{
			name: "with global model",
			routerConfig: &config.RouterConfig{
				ConfigURL: "memory://router-config",
				InputName: "router_id",
				Global: config.GlobalModelConfig{
					Exists: true, // avoid fixed replacements path
				},
			},
			verifier: func(t *testing.T, results []interface{}) {
				if len(results) != 1 {
					t.Fatalf("expected 1 output, got %d", len(results))
				}
				out, ok := results[0].([][]float32)
				if !ok {
					t.Fatalf("expected [][]float32, got %T", results[0])
				}
				want := [][]float32{{1}, {4}}
				if !reflect.DeepEqual(out, want) {
					t.Errorf("output mismatch: got %#v, want %#v", out, want)
				}
			},
		},
		{
			name: "without global model",
			routerConfig: &config.RouterConfig{
				ConfigURL: "memory://router-config",
				InputName: "router_id",
				Global: config.GlobalModelConfig{
					PredictionReplacements: []config.PredictionReplacement{
						{
							Name:  "score",
							Type:  "float32",
							Value: 1.0,
						},
					},
				},
			},
			verifier: func(t *testing.T, results []interface{}) {
				if len(results) != 1 {
					t.Fatalf("expected 1 output, got %d", len(results))
				}
				out, ok := results[0].([][]float32)
				if !ok {
					t.Fatalf("expected [][]float32, got %T", results[0])
				}
				want := [][]float32{{1}, {4}}
				if !reflect.DeepEqual(out, want) {
					t.Errorf("output mismatch: got %#v, want %#v", out, want)
				}
			},
		},
		{
			name: "with model output name",
			routerConfig: &config.RouterConfig{
				ConfigURL: "memory://router-config",
				InputName: "router_id",
				Global: config.GlobalModelConfig{
					Exists: true,
				},
				Output: config.OutputConfig{
					FieldName: "model_output",
				},
			},
			verifier: func(t *testing.T, results []interface{}) {
				if len(results) != 2 {
					t.Fatalf("expected 2 outputs, got %d, %v", len(results), results)
				}

				func() {
					out, ok := results[0].([][]float32)
					if !ok {
						t.Fatalf("expected [][]float32, got %T", results[0])
					}
					want := [][]float32{{1}, {4}}
					if !reflect.DeepEqual(out, want) {
						t.Errorf("output mismatch: got %#v, want %#v", out, want)
					}
				}()

				func() {
					out, ok := results[1].([][]string)
					if !ok {
						t.Fatalf("expected [][]string, got %T", results[1])
					}
					want := [][]string{{"model1"}, {"model2"}}
					if !reflect.DeepEqual(out, want) {
						t.Errorf("output mismatch: got %#v, want %#v", out, want)
					}
				}()
			},
		},
	}

	downstreamInput := domain.Input{
		Name:  "text",
		Index: 1,
		Type:  reflect.TypeOf(""),
	}

	downstreamSignature := &domain.Signature{
		Inputs: []domain.Input{downstreamInput},
		Outputs: []domain.Output{
			{Name: "score", Index: 0, DataType: "float32"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Model{
				ID:       "router_test",
				Mode:     "router",
				Platform: "triton",
				Router:   test.routerConfig,
				Triton: &config.TritonConfig{
					ServerID: "test_server",
				},
			}

			cfg.Init(nil)
			cfg.Router.MaxQueueSize = 1000
			cfg.Router.Workers = 3

			router, err := newRouter(cfg, nil, map[string]UnloadService{
				"test_server": &triton.Service{},
			}, func(modelName string) (platform.PlatformEvaluator, error) {
				return &mockEvaluator{signature: func() *domain.Signature { return downstreamSignature }}, nil
			})

			if err != nil {
				t.Fatalf("NewRouter error: %v", err)
			}

			router.routingMap = map[int]string{
				1: "model1",
				2: "model2",
			}

			routerOutputs := []domain.Output{
				{Name: "score", Index: 0, DataType: "float32"},
			}

			if test.routerConfig.Output.FieldName != "" {
				routerOutputs = append(routerOutputs, domain.Output{Name: test.routerConfig.Output.FieldName, Index: 1, DataType: "string"})
			}

			routerInputName := cfg.Router.InputName
			routerInput := domain.Input{Name: routerInputName, Index: 0, Type: reflect.TypeOf(int64(0))}

			router.ioState = &IOState{
				inputs: map[string]*domain.Input{
					routerInputName:      &routerInput,
					downstreamInput.Name: &downstreamInput,
				},

				signature: &domain.Signature{
					Inputs: []domain.Input{
						routerInput,
						downstreamInput,
					},
					Outputs: routerOutputs,
				},
			}

			mockEval := &mockEvaluator{signature: func() *domain.Signature { return downstreamSignature }}
			router.routingTable = map[string]platform.PlatformEvaluator{
				"model1": mockEval,
				"model2": mockEval,
			}

			// batch of 2
			params := []interface{}{
				[][]int64{{1}, {2}},         // router id
				[][]string{{"a"}, {"abcd"}}, // backend input
			}

			results, err := router.Predict(ctx, params)
			if err != nil {
				t.Fatalf("Predict error: %v", err)
			}

			test.verifier(t, results)
		})
	}
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

type wrappedUnloader struct {
	tritonService *triton.Service

	wg *sync.WaitGroup
}

func (w *wrappedUnloader) UnloadModel(ctx context.Context, mlyModelID string, tritonModelName string) error {
	defer w.wg.Done()
	return w.tritonService.UnloadModel(ctx, mlyModelID, tritonModelName)
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

// --- Batched Prediction Tests ---

// testMockEvaluator is a configurable mock that handles batched inputs and tracks calls
type testMockEvaluator struct {
	modelName    string
	signature    *domain.Signature
	predictCalls int
	mu           sync.Mutex
	err          error // if set, Predict returns this error
}

func newTestMockEvaluator(name string, sig *domain.Signature) *testMockEvaluator {
	return &testMockEvaluator{modelName: name, signature: sig}
}

func (m *testMockEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	m.mu.Lock()
	m.predictCalls++
	m.mu.Unlock()

	if m.err != nil {
		return nil, m.err
	}

	if len(params) == 0 {
		return nil, fmt.Errorf("no params provided")
	}

	// Determine batch size from first input
	var batchSize int
	switch typed := params[0].(type) {
	case [][]string:
		batchSize = len(typed)
	case [][]int32:
		batchSize = len(typed)
	case [][]int64:
		batchSize = len(typed)
	case [][]float32:
		batchSize = len(typed)
	case [][]float64:
		batchSize = len(typed)
	default:
		return nil, fmt.Errorf("unexpected input type: %T", params[0])
	}

	// Compute output: for each row, output the length of the first string input
	results := make([][]float32, batchSize)
	for i := 0; i < batchSize; i++ {
		var length int
		if typed, ok := params[0].([][]string); ok {
			length = len(typed[i][0])
		}
		results[i] = []float32{float32(length)}
	}

	return []interface{}{results}, nil
}

func (m *testMockEvaluator) Signature() *domain.Signature     { return m.signature }
func (m *testMockEvaluator) Dictionary() *common.Dictionary   { return nil }
func (m *testMockEvaluator) Inputs() map[string]*domain.Input { return nil }
func (m *testMockEvaluator) Stats(map[string]interface{})     {}
func (m *testMockEvaluator) Close() error                     { return nil }
func (m *testMockEvaluator) ReloadIfNeeded(ctx context.Context) error {
	return nil
}

func (m *testMockEvaluator) GetPredictCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.predictCalls
}

// predictTestSetup holds the common test setup for prediction tests
type predictTestSetup struct {
	router     *Router
	evaluators map[string]*testMockEvaluator
	signature  *domain.Signature
}

// predictTestOptions configures the test setup
type predictTestOptions struct {
	forceBatchSize1 bool
	modelOutputName string
	modelNames      []string // defaults to ["model1", "model2"]
}

// setupPredictTest creates a router with mock evaluators for prediction testing
func setupPredictTest(t *testing.T, opts predictTestOptions) *predictTestSetup {
	t.Helper()

	if len(opts.modelNames) == 0 {
		opts.modelNames = []string{"model1", "model2"}
	}

	// Create shared signature
	downstreamSig := &domain.Signature{
		Inputs: []domain.Input{
			{Name: "text", Index: 0, Type: reflect.TypeOf("")},
		},
		Outputs: []domain.Output{
			{Name: "score", Index: 0, DataType: "float32"},
		},
	}

	// Create evaluators
	evaluators := make(map[string]*testMockEvaluator)
	for _, name := range opts.modelNames {
		evaluators[name] = newTestMockEvaluator(name, downstreamSig)
	}

	// Create config
	cfg := &config.Model{
		ID:       "router_predict_test",
		Mode:     "router",
		Platform: "triton",
		Router: &config.RouterConfig{
			ConfigURL:       "memory://router-config",
			InputName:       "router_id",
			ForceBatchSize1: opts.forceBatchSize1,
			Global:          config.GlobalModelConfig{Exists: true},
		},
		Triton: &config.TritonConfig{ServerID: "test_server"},
	}

	if opts.modelOutputName != "" {
		cfg.Router.Output.FieldName = opts.modelOutputName
	}

	cfg.Init(nil)
	cfg.Router.MaxQueueSize = 1000
	cfg.Router.Workers = 10

	// Create router
	router, err := newRouter(cfg, nil, map[string]UnloadService{
		"test_server": &triton.Service{},
	}, func(modelName string) (platform.PlatformEvaluator, error) {
		if eval, ok := evaluators[modelName]; ok {
			return eval, nil
		}
		return evaluators[opts.modelNames[0]], nil
	})
	if err != nil {
		t.Fatalf("newRouter error: %v", err)
	}

	// Setup routing map (1->model1, 2->model2, etc.)
	router.routingMap = make(map[int]string)
	for i, name := range opts.modelNames {
		router.routingMap[i+1] = name
	}

	// Setup routing table
	router.routingTable = make(map[string]platform.PlatformEvaluator)
	for name, eval := range evaluators {
		router.routingTable[name] = eval
	}

	// Build signature outputs
	outputs := []domain.Output{{Name: "score", Index: 0, DataType: "float32"}}
	if opts.modelOutputName != "" {
		outputs = append(outputs, domain.Output{Name: opts.modelOutputName, Index: 1, DataType: "string"})
	}

	// Setup ioState
	routerInputName := cfg.Router.InputName
	routerInput := domain.Input{Name: routerInputName, Index: 1, Type: reflect.TypeOf(int64(0))}
	downstreamInput := domain.Input{Name: "text", Index: 0, Type: reflect.TypeOf("")}

	routerSig := &domain.Signature{
		Inputs:  []domain.Input{downstreamInput, routerInput},
		Outputs: outputs,
	}

	router.ioState = &IOState{
		inputs: map[string]*domain.Input{
			routerInputName:      &routerInput,
			downstreamInput.Name: &downstreamInput,
		},
		signature:         routerSig,
		routerInputOffset: 1,
	}

	return &predictTestSetup{
		router:     router,
		evaluators: evaluators,
		signature:  routerSig,
	}
}

func TestRouter_Predict_BatchingBehavior(t *testing.T) {
	tests := []struct {
		name            string
		forceBatchSize1 bool
		inputs          []string
		routingIDs      []int64
		wantScores      [][]float32
		wantCallCounts  map[string]int
	}{
		{
			name:            "batched_groups_by_model",
			forceBatchSize1: false,
			inputs:          []string{"a", "bb", "ccc", "dddd", "eeeee", "ffffff"},
			routingIDs:      []int64{1, 2, 1, 2, 1, 2}, // alternating model1/model2
			wantScores:      [][]float32{{1}, {2}, {3}, {4}, {5}, {6}},
			wantCallCounts:  map[string]int{"model1": 1, "model2": 1}, // each model called once
		},
		{
			name:            "batched_single_model",
			forceBatchSize1: false,
			inputs:          []string{"a", "bb", "ccc", "dddd"},
			routingIDs:      []int64{1, 1, 1, 1}, // all to model1
			wantScores:      [][]float32{{1}, {2}, {3}, {4}},
			wantCallCounts:  map[string]int{"model1": 1, "model2": 0},
		},
		{
			name:            "force_batch_size_1",
			forceBatchSize1: true,
			inputs:          []string{"a", "bb", "ccc", "dddd"},
			routingIDs:      []int64{1, 1, 1, 1}, // all to model1
			wantScores:      [][]float32{{1}, {2}, {3}, {4}},
			wantCallCounts:  map[string]int{"model1": 4, "model2": 0}, // 4 individual calls
		},
		{
			name:            "force_batch_size_1_multiple_models",
			forceBatchSize1: true,
			inputs:          []string{"a", "bb", "ccc", "dddd"},
			routingIDs:      []int64{1, 2, 1, 2}, // alternating
			wantScores:      [][]float32{{1}, {2}, {3}, {4}},
			wantCallCounts:  map[string]int{"model1": 2, "model2": 2}, // 2 calls each
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := setupPredictTest(t, predictTestOptions{
				forceBatchSize1: tt.forceBatchSize1,
			})

			// Build input params
			stringInputs := make([][]string, len(tt.inputs))
			for i, s := range tt.inputs {
				stringInputs[i] = []string{s}
			}
			routingInputs := make([][]int64, len(tt.routingIDs))
			for i, id := range tt.routingIDs {
				routingInputs[i] = []int64{id}
			}

			params := []interface{}{stringInputs, routingInputs}

			results, err := setup.router.Predict(context.Background(), params)
			if err != nil {
				t.Fatalf("Predict error: %v", err)
			}

			// Verify scores
			scores, ok := results[0].([][]float32)
			if !ok {
				t.Fatalf("expected [][]float32, got %T", results[0])
			}
			if !reflect.DeepEqual(scores, tt.wantScores) {
				t.Errorf("scores mismatch:\n  got:  %v\n  want: %v", scores, tt.wantScores)
			}

			// Verify call counts
			for modelName, wantCount := range tt.wantCallCounts {
				if eval, ok := setup.evaluators[modelName]; ok {
					gotCount := eval.GetPredictCalls()
					if gotCount != wantCount {
						t.Errorf("%s call count: got %d, want %d", modelName, gotCount, wantCount)
					}
				}
			}
		})
	}
}

func TestRouter_Predict_WithModelOutput(t *testing.T) {
	setup := setupPredictTest(t, predictTestOptions{
		modelOutputName: "model_id",
	})

	params := []interface{}{
		[][]string{{"a"}, {"bb"}, {"ccc"}, {"dddd"}},
		[][]int64{{1}, {2}, {1}, {2}},
	}

	results, err := setup.router.Predict(context.Background(), params)
	if err != nil {
		t.Fatalf("Predict error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 outputs, got %d", len(results))
	}

	// Verify scores
	scores, ok := results[0].([][]float32)
	if !ok {
		t.Fatalf("expected [][]float32, got %T", results[0])
	}
	wantScores := [][]float32{{1}, {2}, {3}, {4}}
	assert.Equal(t, wantScores, scores, "scores mismatch")

	// Verify model IDs
	modelIDs, ok := results[1].([][]string)
	if !ok {
		t.Fatalf("expected [][]string for model_id, got %T", results[1])
	}
	wantModelIDs := [][]string{{"model1"}, {"model2"}, {"model1"}, {"model2"}}
	assert.Equal(t, wantModelIDs, modelIDs, "model_id mismatch")
}

func TestRouter_Predict_ErrorPropagation(t *testing.T) {
	setup := setupPredictTest(t, predictTestOptions{
		modelNames: []string{"model1"},
	})

	// Configure evaluator to return error
	setup.evaluators["model1"].err = errors.New("model prediction failed")

	params := []interface{}{
		[][]string{{"a"}, {"bb"}},
		[][]int64{{1}, {1}},
	}

	_, err := setup.router.Predict(context.Background(), params)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !strings.Contains(err.Error(), "model prediction failed") {
		t.Errorf("expected error to contain 'model prediction failed', got: %v", err)
	}
}
