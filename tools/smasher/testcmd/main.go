package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/viant/mly/tools/smasher"
)

type cl struct {
	cli  *http.Client
	URL  string
	sent uint64
}

func (c *cl) Do() error {
	_, err := c.cli.Get(c.URL)
	atomic.AddUint64(&c.sent, 1)
	return err
}

func (c *cl) Sent() uint64 {
	return c.sent
}

func main() {
	sleeper := new(smasher.Sleeper)
	sleeper.D = 1 * time.Second

	ls := new(smasher.Lambda)
	ls.H = sleeper

	port := 8889
	srv := new(http.Server)
	srv.Addr = fmt.Sprintf(":%d", port)
	srv.Handler = ls

	var serverErr error
	go func() {
		serverErr = srv.ListenAndServe()
		if serverErr != nil && serverErr != http.ErrServerClosed {
			log.Fatal(serverErr)
		}
	}()

	smasher.Run(smasher.TestStruct{
		Server: func() smasher.Server {
			return ls
		},
		Client: func() smasher.Client {
			c := new(cl)
			c.URL = fmt.Sprintf("http://localhost:%d", port)
			c.cli = new(http.Client)
			return c
		},
	}, -1, 150000, 1*time.Second)
}
