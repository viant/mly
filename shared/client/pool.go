package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/viant/mly/shared/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"strings"
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

var certPool *x509.CertPool

func newConn(addr string) (*grpcClient, error) {
	var options = make([]grpc.DialOption, 0)
	var err error
	if strings.HasSuffix(addr, ":443") {
		if certPool == nil {
			certPool, err = x509.SystemCertPool()
			if err != nil {
				return nil, fmt.Errorf("failed to create certificate")
			}
		}
		config := &tls.Config{
			RootCAs: certPool,
		}
		options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(config)))
	} else {
		options = append(options, grpc.WithInsecure())
	}
	conn, err := grpc.Dial(addr,options...)
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
		if p.err == nil {
			p.err = fmt.Errorf("failed to create client")
		}
		return nil, p.err
	}
	if atomic.AddInt32(&p.current, -1) < 0 {
		atomic.AddInt32(&p.current, 1)
	}
	return result.(*grpcClient), nil
}

//Reset reset pooled connection
func (p *grpcPool) Reset() {
	for atomic.AddInt32(&p.current, -1) >= 0 {
		if result := p.Pool.Get(); result != nil {
			conn := result.(*grpc.ClientConn)
			_ = conn.Close()
		}
	}
}

func newGrpcPool(maxSize int, addr string) *grpcPool {
	result := &grpcPool{max: int32(maxSize)}
	result.Pool.New = func() interface{} {
		cl, err := newConn(addr)
		if err != nil {
			result.err = err
			return nil
		}
		if cl != nil {
			cl.pool = result
		}
		return cl
	}
	return result
}
