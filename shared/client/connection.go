package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	dialTCPFragment  = "dial tcp"
	connRefusedError = "refused"
)

var requestTimeout = 5 * time.Second

type connection struct {
	url string
	*io.PipeWriter
	*io.PipeReader
	req    *http.Request
	resp   *http.Response
	buf    []byte
	ctx    context.Context
	cancel context.CancelFunc
	closed int32
	host   *Host
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
	fmt.Println("closing conn")
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		if c.req != nil && c.req.Body != nil {
			c.req.Body.Close()
		}
		if c.PipeWriter != nil {
			c.PipeWriter.Close()
		}
		if c.PipeReader != nil {
			c.PipeReader.Close()
		}
		if c.resp != nil && c.resp.Body != nil {
			c.resp.Body.Close()
		}
		if c.cancel != nil {
			c.cancel()
		}
	}
	return nil
}

func newConnection(host *Host, httpClient *http.Client, URL string) (*connection, error) {
	ctx, cancel := context.WithCancel(context.Background())
	result := &connection{
		buf:    make([]byte, 16*1024),
		host:   host,
		ctx:    ctx,
		cancel: cancel,
	}
	return result, result.init(httpClient, URL)
}

func (c *connection) init(httpClient *http.Client, URL string) error {
	c.PipeReader, c.PipeWriter = io.Pipe()
	req, err := http.NewRequestWithContext(c.ctx, http.MethodPut, URL, c.PipeReader)
	if err != nil {
		if strings.Contains(err.Error(), dialTCPFragment) || strings.Contains(err.Error(), connRefusedError) {
			c.host.FlagDown()
		}
		_ = c.Close()
		return err
	}
	c.req = req
	c.resp, err = httpClient.Do(req)
	if err != nil {
		_ = c.Close()
	}
	return err
}
