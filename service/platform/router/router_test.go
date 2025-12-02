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

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/platform"
	"github.com/viant/mly/shared/common"
	sharedrouter "github.com/viant/mly/shared/config/router"
)

// --- Router Predict scaffolds ---

type mockTritonServer struct {
	mu sync.Mutex

	readyState   map[string]bool
	modelLoadErr map[string]error
}

type mockEvaluator struct {
	tritonServer *mockTritonServer

	modelName string
	signature *domain.Signature
}

func (m *mockEvaluator) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
	inputs := m.signature.Inputs
	if len(inputs) != len(params) {
		return nil, fmt.Errorf("expected %d inputs, got %d", len(inputs), len(params))
	}

	// params is expected to have a single non-router input in this test: [][]string with shape [1][1]
	var v string
	switch typed := params[0].(type) {
	case [][]string:
		v = typed[0][0]
	default:
		tval := reflect.TypeOf(params[0])
		panic("unexpected input type in mockPredictOnly: " + tval.String())
	}
	// simple function: length of string as float32
	out := [][]float32{{float32(len(v))}}
	return []interface{}{out}, nil
}

func (m *mockEvaluator) Signature() *domain.Signature     { return m.signature }
func (m *mockEvaluator) Dictionary() *common.Dictionary   { return nil }
func (m *mockEvaluator) Inputs() map[string]*domain.Input { return nil }
func (m *mockEvaluator) Stats(map[string]interface{})     {}
func (m *mockEvaluator) Close() error                     { return nil }

func (m *mockEvaluator) ReloadIfNeeded(ctx context.Context) error {
	if m.tritonServer == nil {
		return nil
	}

	m.tritonServer.mu.Lock()
	defer m.tritonServer.mu.Unlock()

	if err := m.tritonServer.modelLoadErr[m.modelName]; err != nil {
		return err
	}

	if m.tritonServer.readyState == nil {
		m.tritonServer.readyState = make(map[string]bool)
	}

	m.tritonServer.readyState[m.modelName] = true
	return nil
}

type mockUnloader struct {
	tritonServer *mockTritonServer
	unloadCh     chan string
}

func (m *mockUnloader) ModelUnload(ctx context.Context, modelName string) error {
	if m.tritonServer != nil {
		m.tritonServer.mu.Lock()
		defer m.tritonServer.mu.Unlock()

		if m.tritonServer.readyState == nil {
			m.tritonServer.readyState = make(map[string]bool)
		}

		m.tritonServer.readyState[modelName] = false
	}

	ch := m.unloadCh

	if ch != nil {
		ch <- modelName
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

			router, err := newRouter(cfg, nil, map[string]ModelUnloader{
				"test_server": &mockUnloader{},
			}, func(modelName string) (platform.PlatformEvaluator, error) {
				return &mockEvaluator{signature: downstreamSignature}, nil
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

			mockEval := &mockEvaluator{signature: downstreamSignature}
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

	oldConfig := &sharedrouter.RouterConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelA"},
			{EntityID: 2, ModelName: "modelB"},
		},
		GlobalModelName: "global-old",
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
		unloader:     mockClient,
		routerConfig: oldConfig,
		routingMap: map[int]string{
			1: "modelA",
			2: "modelB",
		},
		routingTable: map[string]platform.PlatformEvaluator{
			"modelA": &mockEvaluator{signature: signature},
			"modelB": &mockEvaluator{signature: signature},
		},
		globalModel: &mockEvaluator{},
		debug:       true,
		makeRoutedEvaluator: func(modelName string) (platform.PlatformEvaluator, error) {
			return &mockEvaluator{signature: signature}, nil
		},
	}

	reusedModelB := router.routingTable["modelB"]

	newConfig := &sharedrouter.RouterConfig{
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

	if router.routerConfig != newConfig {
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

	oldConfig := &sharedrouter.RouterConfig{
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

		unloader:     mockClient,
		routerConfig: oldConfig,
		routingMap: map[int]string{
			1: "modelA",
		},

		makeRoutedEvaluator: func(modelName string) (platform.PlatformEvaluator, error) {
			return &mockEvaluator{
				tritonServer: tritonServer,
				modelName:    modelName,
				signature:    signature,
			}, nil
		},
	}

	newConfig := &sharedrouter.RouterConfig{
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

	if router.routerConfig != oldConfig {
		t.Fatalf("routerConfig should remain unchanged on error")
	}

	if !reflect.DeepEqual(router.routingMap, map[int]string{1: "modelA"}) {
		t.Fatalf("routingMap should remain unchanged on error")
	}

	if router.routingTable != nil {
		t.Fatalf("routingTable should not be replaced on error")
	}
}

func TestRouter_applyRouterConfig_SkipsLoadWhenReady(t *testing.T) {
	ctx := context.Background()
	mockClient := &mockUnloader{}

	oldConfig := &sharedrouter.RouterConfig{
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
		unloader:     mockClient,
		routerConfig: oldConfig,
		routingMap: map[int]string{
			1: "modelA",
		},
		routingTable: map[string]platform.PlatformEvaluator{
			"modelA": &mockEvaluator{signature: signature},
		},
		makeRoutedEvaluator: func(modelName string) (platform.PlatformEvaluator, error) {
			return &mockEvaluator{signature: signature}, nil
		},
	}

	newConfig := &sharedrouter.RouterConfig{
		EntityMapping: []sharedrouter.EntityKV{
			{EntityID: 1, ModelName: "modelA"},
			{EntityID: 2, ModelName: "modelC"},
		},
	}

	if err := router.applyRouterConfig(ctx, newConfig); err != nil {
		t.Fatalf("applyRouterConfig returned error: %v", err)
	}

	if _, ok := router.routingTable["modelC"]; !ok {
		t.Fatalf("routingTable missing modelC after reload")
	}
}
