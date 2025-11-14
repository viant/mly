package router

import (
	"context"
	"reflect"
	"testing"

	"github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	"github.com/viant/mly/service/platform"
	tricli "github.com/viant/mly/service/triton"
	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/common"
)

// --- Router Predict scaffolds ---

type mockPredictOnly struct{}

func (m *mockPredictOnly) Predict(ctx context.Context, params []interface{}) ([]interface{}, error) {
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

func (m *mockPredictOnly) Signature() *domain.Signature     { return nil }
func (m *mockPredictOnly) Dictionary() *common.Dictionary   { return nil }
func (m *mockPredictOnly) Inputs() map[string]*domain.Input { return nil }
func (m *mockPredictOnly) Stats(map[string]interface{})     {}
func (m *mockPredictOnly) Close() error                     { return nil }
func (m *mockPredictOnly) ReloadIfNeeded(ctx context.Context) error {
	return nil
}

type mockTritonClient struct{}

func (m *mockTritonClient) ServerReady(ctx context.Context) error { return nil }
func (m *mockTritonClient) ModelInfer(ctx context.Context, modelName string, inputs []interface{}, indexToName map[int]string) ([]interface{}, error) {
	return nil, nil
}
func (m *mockTritonClient) ModelReady(ctx context.Context, modelName string) (bool, error) {
	return true, nil
}
func (m *mockTritonClient) ModelLoad(ctx context.Context, modelName string) error   { return nil }
func (m *mockTritonClient) ModelUnload(ctx context.Context, modelName string) error { return nil }
func (m *mockTritonClient) Close() error                                            { return nil }

func TestRouter_Predict_RoutesAndConcats(t *testing.T) {
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
					Exists: true, // avoid fixed replacements path
				},
				Output: config.OutputConfig{
					FieldName: "model_output",
				},
			},
			verifier: func(t *testing.T, results []interface{}) {
				if len(results) != 2 {
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
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			cfg := &config.Model{
				ID:       "router_test",
				Mode:     "router",
				Platform: "triton",
				MetaInput: shared.MetaInput{
					Inputs: []*shared.Field{
						// router input first (default offset 0)
						{Name: "router_id", Index: 0, DataType: "int64"},
						// single backend input
						{Name: "text", Index: 1, DataType: "string"},
					},
					Outputs: []*shared.Field{
						{Name: "score", Index: 0, DataType: "float32"},
					},
				},
				Router: test.routerConfig,
				Triton: &config.TritonConfig{
					ServerID: "test_server",
				},
			}

			router, err := NewRouter(cfg, nil, map[string]tricli.TritonClient{
				"test_server": &mockTritonClient{},
			})

			if err != nil {
				t.Fatalf("NewRouter error: %v", err)
			}

			router.routingMap = map[int]string{
				1: "model1",
				2: "model2",
			}
			mockEval := &mockPredictOnly{}
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

func TestConcatAxis0_Int32(t *testing.T) {
	x := []interface{}{
		[][]int32{{1}, {2}},
	}
	y := []interface{}{
		[][]int32{{3}},
	}

	got, err := concatAxis0(x, y)
	if err != nil {
		t.Fatalf("concatAxis0 returned error: %v", err)
	}

	want := []interface{}{
		[][]int32{{1}, {2}, {3}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("concatAxis0() mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestConcatAxis0_String(t *testing.T) {
	x := []interface{}{
		[][]string{{"a"}},
	}
	y := []interface{}{
		[][]string{{"b"}, {"c"}},
	}

	got, err := concatAxis0(x, y)
	if err != nil {
		t.Fatalf("concatAxis0 returned error: %v", err)
	}

	want := []interface{}{
		[][]string{{"a"}, {"b"}, {"c"}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("concatAxis0() mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestConcatAxis0_TypeMismatch(t *testing.T) {
	x := []interface{}{
		[][]int64{{1}},
	}
	y := []interface{}{
		[][]int32{{2}},
	}

	_, err := concatAxis0(x, y)
	if err == nil {
		t.Fatalf("expected type mismatch error, got nil")
	}
}

func TestDebatchAndSqueezeBatch_Int64(t *testing.T) {
	// batch of 3, single-column
	batch := [][]int64{{10}, {20}, {30}}

	// pick index 1
	debatched, err := debatch(batch, 1)
	if err != nil {
		t.Fatalf("debatch returned error: %v", err)
	}

	wantDebatched := [][]int64{{20}}
	if !reflect.DeepEqual(debatched, wantDebatched) {
		t.Errorf("debatch() mismatch:\n got: %#v\nwant: %#v", debatched, wantDebatched)
	}

	// squeeze should return the scalar 20
	scalar, err := squeezeBatch(debatched)
	if err != nil {
		t.Fatalf("squeezeBatch returned error: %v", err)
	}

	if v, ok := scalar.(int64); !ok || v != 20 {
		t.Errorf("squeezeBatch() got %v (%T), want 20 (int64)", scalar, scalar)
	}
}
