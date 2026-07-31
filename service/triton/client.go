package triton

import (
	"context"
	"net/http"
	"time"

	grpcProm "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/viant/mly/service/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

// A TritonClient represents a client to a single Triton server.
type TritonClient interface {
	ServerReady(ctx context.Context) error

	// inputs is expected to be [numInputs]([batchSize][1]T) (see service/request.Request.Feeds)
	// inputs will never be empty
	//
	// Returns the output tensors keyed by their Triton output tensor name. Callers
	// (the evaluator) are responsible for mapping these into signature order; the
	// map deliberately carries no positional/order information so no consumer can
	// depend on the order Triton happens to return tensors in.
	ModelInfer(ctx context.Context, modelName string, inputs []interface{}, indexToName map[int]string) (map[string]interface{}, error)

	ModelReady(ctx context.Context, modelName string) (bool, error)

	ModelLoad(ctx context.Context, modelName string) error

	ModelUnloader

	ModelMetadata(ctx context.Context, modelName string) (*ModelMetadata, error)

	Close() error
}

type ModelUnloader interface {
	ModelUnload(ctx context.Context, modelName string) error
}

// https://github.com/kserve/kserve/blob/master/docs/predict-api/v2/required_api.md#model-metadata-response-json-object `$metadata_tensor`
type MetadataTensor struct {
	Name     string  `json:"name"`
	Datatype string  `json:"datatype"`
	Shape    []int64 `json:"shape"`
}

// stripped down version of https://github.com/kserve/kserve/blob/master/docs/predict-api/v2/required_api.md#model-metadata-response-json-object
type ModelMetadata struct {
	Inputs  []MetadataTensor `json:"inputs"`
	Outputs []MetadataTensor `json:"outputs"`
}

// NewClient creates either an HTTP or gRPC client.
func NewClient(server config.TritonServer) (TritonClient, error) {
	if server.GRPCBaseURL != "" {
		grpcConn, err := grpc.NewClient(server.GRPCBaseURL,
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff: backoff.Config{
					BaseDelay:  time.Duration(server.GRPCConnectParams.BaseDelayMs) * time.Millisecond,
					Multiplier: server.GRPCConnectParams.Multiplier,
					Jitter:     server.GRPCConnectParams.Jitter,
					MaxDelay:   time.Duration(server.GRPCConnectParams.MaxDelayMs) * time.Millisecond,
				},
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),

			grpc.WithUnaryInterceptor(grpcProm.UnaryClientInterceptor),
			grpc.WithStreamInterceptor(grpcProm.StreamClientInterceptor),
		)

		if err != nil {
			return nil, err
		}

		return NewMeteredTritonClient(NewGRPCClient(grpcConn)), nil
	}

	// HTTP options seem a bit bare
	return NewMeteredTritonClient(&HTTPClient{
		httpClient: &http.Client{
			Timeout: time.Duration(server.HTTPClientTimeoutMs) * time.Millisecond,
		},
		serverURL: server.HTTPBaseURL,
	}), nil
}
