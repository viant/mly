package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/viant/mly/service/endpoint"
	"github.com/viant/mly/shared/client"
	"github.com/viant/mly/tools/smasher"
)

type mlyS struct {
	cli        *http.Client
	metricPath string
	ma         uint64
	l          sync.Mutex
}

type stats struct {
	Count    int
	Counters []counter
}

type counter struct {
	Value string
	Count int
}

func (s *mlyS) Stats() string {
	stp := s.getStats("Perf")
	ste := s.getStats("Eval")

	var pErr, eErr string
	if len(stp.Counters) > 0 {
		pErr = "noPerf"
	}

	if len(ste.Counters) > 0 {
		eErr = "noEval"
	}

	var perfPending, eval int
	for _, c := range stp.Counters {
		if c.Value == "pending" {
			perfPending = c.Count
		}

		if c.Value == "eval" {
			eval = c.Count
		}
	}

	func() {
		s.l.Lock()
		defer s.l.Unlock()
		if uint64(perfPending) > s.ma {
			s.ma = uint64(perfPending)
		}
	}()

	var evalPending int
	for _, c := range ste.Counters {
		if c.Value == "pending" {
			evalPending = c.Count
		}

	}

	return fmt.Sprintf("(%s,%s) perfPending:%d evalPending:%d eval:%d", pErr, eErr, perfPending, evalPending, eval)
}

func (a *mlyS) getStats(suffix string) *stats {
	s := new(stats)
	path := fmt.Sprintf("%s%s", a.metricPath, suffix)
	resp, err := a.cli.Get(path)
	if err != nil {
		return s
	}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(s)
	return s
}

type mlyC struct {
	pool    *sync.Pool
	cli     *client.Service
	timeout time.Duration
	sent    uint64
}

type either struct {
	o *client.Service
	e error
}

type modelResp struct {
	Output float32 `json:"output"`
}

func (c *mlyC) Do() error {
	var err error

	eo := c.pool.Get().(*either)
	cli := eo.o
	err = eo.e
	if err != nil {
		return err
	}

	defer c.pool.Put(eo)

	msg := cli.NewMessage()
	defer msg.Release()

	msg.FloatKey("x", 5.231)
	msg.FloatKey("y", 6.748)

	resp := new(client.Response)
	resp.Data = new(modelResp)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	atomic.AddUint64(&c.sent, 1)
	err = cli.Run(ctx, msg, resp)

	return err
}

func (c *mlyC) Sent() uint64 {
	return c.sent
}

func main() {
	srvWait := new(sync.WaitGroup)
	srvWait.Add(1)

	ctx := context.Background()

	statD := 1 * time.Second

	var mlyPort int
	var serverErr error
	go func() {
		serverErr = endpoint.RunAppWithConfigWaitError("smasher", os.Args[1:], func(options *endpoint.Options) (*endpoint.Config, error) {
			config, err := endpoint.NewConfigFromURL(ctx, options.ConfigURL)
			mlyPort = config.Endpoint.Port
			log.Printf("%d", config.Endpoint.MaxEvaluatorConcurrency)
			return config, err
		}, srvWait)

		log.Printf("%v", serverErr)
	}()

	srvWait.Wait()

	smasher.Run(smasher.TestStruct{
		Server: func() (smasher.Server, error) {
			s := &mlyS{
				cli: &http.Client{
					Timeout: statD,
				},
				metricPath: fmt.Sprintf("http://localhost:%d/v1/api/metric/operation/slow", mlyPort),
			}
			return s, nil
		},
		Client: func() (smasher.Client, error) {
			c := new(mlyC)

			hosts := []*client.Host{client.NewHost("localhost", mlyPort)}

			c.pool = &sync.Pool{
				New: func() interface{} {
					c, e := client.New("slow", hosts, client.WithDebug(false))
					return &either{c, e}
				},
			}

			cli, err := client.New("slow", hosts, client.WithDebug(false))
			if err != nil {
				return nil, err
			}

			c.cli = cli
			c.timeout = 30 * time.Second

			return c, nil

		},
	}, 3000, 20000, statD)
}
