package client

import (
	"context"
	"crypto/tls"
	"golang.org/x/net/http2"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	dialTCPFragment         = "dial tcp"
	connRefusedError        = "refused"

)

var requestTimeout = time.Second

type connection struct {
	url string
	http.Client
	*io.PipeWriter
	req      *http.Request
	resp     *http.Response
	buf      []byte
	lastUsed time.Time
	ctx      context.Context
	closed   int32
	host *Host
}

func (c *connection) Write(data []byte) (int, error) {
	if c.ctx.Err() != nil {
		_ = c.Close()
		return 0, c.ctx.Err()
	}
	size, err := c.PipeWriter.Write(data)
	return size, err
}

func (c *connection) Read() ([]byte, error) {
	if c.ctx.Err() != nil {
		_ = c.Close()
		return nil, c.ctx.Err()
	}
	size, err := c.resp.Body.Read(c.buf)
	if err != nil {
		return nil, err
	}
	return c.buf[:size], nil
}

func (c *connection) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		if c.req != nil && c.req.Body != nil {
			return c.req.Body.Close()
		}
	}
	return nil
}

func newConnection(host *Host, URL string) (*connection, error) {
	result := &connection{
		buf:  make([]byte, 32*1024),
		host: host,
		ctx:  context.Background(),
	}
	return result, result.init(URL)
}

func (c *connection) init(URL string) error {
	c.Transport = &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
	}
	c.Timeout = time.Second
	var pr *io.PipeReader
	pr, c.PipeWriter = io.Pipe()
	req, err := http.NewRequestWithContext(c.ctx, http.MethodPut, URL, pr)
	if err != nil {
		if strings.Contains(err.Error(), dialTCPFragment) ||strings.Contains(err.Error(), connRefusedError) {
			c.host.FlagDown()
		}
		return err
	}

	c.req = req
	c.resp, err = c.Client.Do(req)
	return err
}

