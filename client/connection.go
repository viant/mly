package client

import (
	"crypto/tls"
	"golang.org/x/net/http2"
	"io"
	"net"
	"net/http"
	"time"
)

type connection struct {
	url string
	http.Client
	*io.PipeWriter
	req *http.Request
	resp *http.Response
	buf []byte
	lastUsed time.Time
}

func (c *connection) Write(data []byte) (int, error) {
	return c.PipeWriter.Write(data)
}

func (c *connection) Read() ([]byte, error) {
	size, err := c.resp.Body.Read(c.buf)
	if err != nil {
		return nil, err
	}
	return c.buf[:size], nil
}

func (c *connection) Close() error {
	if c.req != nil && c.req.Body != nil {
		return c.req.Body.Close()
	}
	return nil
}

func newConnection(URL string) (*connection, error) {
	result := &connection{
		buf: make([]byte, 32*1024),
	}
	result.Transport = &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
	}
	result.Timeout = time.Second
	var pr *io.PipeReader
	pr, result.PipeWriter = io.Pipe()
	req, err := http.NewRequest(http.MethodPut, URL, pr)
	if err != nil {
		return nil, err
	}
	result.req = req
	result.resp, err = result.Client.Do(req)
	if err != nil {
		return nil, err
	}
	return result, nil
}
