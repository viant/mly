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

type TritonClient interface {
	ServerReady(ctx context.Context) error

	// inputs is expected to be [numInputs]([batchSize][1]T) (see service/request.Request.Feeds)
	// inputs will never be empty
	ModelInfer(ctx context.Context, modelName string, inputs []interface{}, indexToName map[int]string) ([]interface{}, error)

	ModelReady(ctx context.Context, modelName string) (bool, error)

	ModelLoad(ctx context.Context, modelName string) error

	ModelUnload(ctx context.Context, modelName string) error

	Close() error
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
	// TODO see if DRY with
	return NewMeteredTritonClient(&HTTPClient{
		httpClient: &http.Client{
			Timeout: time.Duration(server.HTTPClientTimeoutMs) * time.Millisecond,
		},
		serverURL: server.HTTPBaseURL,
	}), nil
}
