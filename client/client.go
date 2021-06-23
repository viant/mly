package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/francoispqt/gojay"
	"github.com/viant/mly/client/config"
	"github.com/viant/toolbox"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	Config
	pool      sync.Pool
	poolErr   error
	hostIndex int64
}

func (c *Client) conn() (*connection, error) {
	result := c.pool.Get()
	if result == nil {
		return nil, c.poolErr
	}
	conn := result.(*connection)
	if conn.lastUsed.IsZero() {
		return conn, nil
	}
	if time.Now().Sub(conn.lastUsed) > conn.Timeout {
		_ = conn.Close()
		return c.conn()
	}
	return conn, nil
}

//Run run model prediction
func (c *Client) Run(ctx context.Context, input interface{}, response *Response) error {
	data, err := NewReader(input)
	if err != nil {
		return err
	}
	conn, err := c.conn()
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	if err != nil {
		return err
	}
	body, err := conn.Read()
	if err != nil {
		return err
	}
	conn.lastUsed = time.Now()
	c.pool.Put(conn)
	err = gojay.Unmarshal(body, response)
	if err != nil {
		return fmt.Errorf("failed to unmarshal: '%s'; due to %w", body, err)
	}
	return nil
}

func (c *Client) getHost() *Host {
	count := len(c.Hosts)
	switch count {
	case 1:
		return c.Hosts[0]
	default:
		index := atomic.AddInt64(&c.hostIndex, 1) % int64(count)
		return c.Hosts[index]
	}
}

func (c *Client) evalURL(model string) string {
	return c.getHost().evalURL(model)
}

func (c *Client) metaConfigURL(model string) string {
	return c.getHost().metaConfigURL(model)
}

func (c *Client) metaDictionaryURL(model string) string {
	return c.getHost().metaDictionaryURL(model)
}


//NewClient creates new mly client
func NewClient(model string, hosts []*Host, options ...Option) (*Client, error) {
	config := Config{
		Model: model,
		Hosts: hosts,
	}
	result := &Client{
		Config: config,
	}

	for _, option := range options {
		option.Apply(result)
	}

	if result.Datastore == nil {
		var err error
		if result.Datastore, err = discoverConfig(result.metaConfigURL(model));err != nil {
			return nil, err
		}
	}

	result.pool.New = func() interface{} {
		conn, err := newConnection(result.evalURL(model))
		if err != nil {
			result.poolErr = err
		}
		return conn
	}
	toolbox.Dump(result.Config)
	return result, nil
}

func discoverConfig(URL string) (*config.Datastore, error) {
	response, err := http.DefaultClient.Get(URL)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	cfg := &config.Datastore{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %s, %v", data, err)
	}
	return cfg, err
}

