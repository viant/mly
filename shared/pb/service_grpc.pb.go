// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// EvaluatorClient is the client API for Evaluator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EvaluatorClient interface {
	// Sends a greeting
	Evaluate(ctx context.Context, in *EvaluateRequest, opts ...grpc.CallOption) (*EvaluateResponse, error)
}

type evaluatorClient struct {
	cc grpc.ClientConnInterface
}

func NewEvaluatorClient(cc grpc.ClientConnInterface) EvaluatorClient {
	return &evaluatorClient{cc}
}

func (c *evaluatorClient) Evaluate(ctx context.Context, in *EvaluateRequest, opts ...grpc.CallOption) (*EvaluateResponse, error) {
	out := new(EvaluateResponse)
	err := c.cc.Invoke(ctx, "/Evaluator/Evaluate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EvaluatorServer is the server API for Evaluator service.
// All implementations must embed UnimplementedEvaluatorServer
// for forward compatibility
type EvaluatorServer interface {
	// Sends a greeting
	Evaluate(context.Context, *EvaluateRequest) (*EvaluateResponse, error)
}

// UnsafeEvaluatorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EvaluatorServer will
// result in compilation errors.
type UnsafeEvaluatorServer interface {
	mustEmbedUnimplementedEvaluatorServer()
}

func RegisterEvaluatorServer(s grpc.ServiceRegistrar, srv EvaluatorServer) {
	s.RegisterService(&Evaluator_ServiceDesc, srv)
}

func _Evaluator_Evaluate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EvaluateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EvaluatorServer).Evaluate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Evaluator/Evaluate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EvaluatorServer).Evaluate(ctx, req.(*EvaluateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Evaluator_ServiceDesc is the grpc.ServiceDesc for Evaluator service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Evaluator_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Evaluator",
	HandlerType: (*EvaluatorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Evaluate",
			Handler:    _Evaluator_Evaluate_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/service.proto",
}