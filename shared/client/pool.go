package client

import (
	"github.com/viant/mly/shared/pb"
	"google.golang.org/grpc"
	"sync"
	"sync/atomic"
)

type grpcClient struct {
	*grpc.ClientConn
	pb.EvaluatorClient
	pool    *grpcPool
	pending int32
}


func (c *grpcClient) Close() {
	c.ClientConn.Close()
}


func (c *grpcClient) Release() {
	pool := c.pool
	if pool == nil {
		c.ClientConn.Close()
		return
	}
	pool.Put(c)
}

func newConn(addr string) (*grpcClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &grpcClient{
		ClientConn:      conn,
		EvaluatorClient: pb.NewEvaluatorClient(conn),
	}, nil
}

type grpcPool struct {
	current int32
	max     int32
	sync.Pool
	err error
}

func (p *grpcPool) Put(client *grpcClient) {
	if atomic.AddInt32(&p.current, 1) < p.max {
		p.Pool.Put(client)
	}
	atomic.AddInt32(&p.current, -1)
}


func (p *grpcPool) Conn() (*grpcClient, error) {
	result := p.Pool.Get()
	if result == nil {
		return nil, p.err
	}
	if atomic.AddInt32(&p.current, -1) < 0 {
		atomic.AddInt32(&p.current, 1)
	}
	return result.(*grpcClient), nil
}


func newGrpcPool(maxSize int, addr string) *grpcPool {
	result := &grpcPool{max: int32(maxSize)}
	result.Pool.New = func() interface{} {
		var cl *grpcClient
		cl, result.err = newConn(addr)
		if cl != nil {
			cl.pool = result
		}
		return cl
	}
	return result
}
