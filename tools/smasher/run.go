package smasher

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/viant/mly/shared/semaph"
)

type (
	Server interface {
		Stats() string
	}

	Client interface {
		Do() error

		Sent() uint64
	}

	TestStruct struct {
		Server func() Server
		Client func() Client
	}
)

func Run(ts TestStruct, maxDos int32, testCases int, statDur time.Duration) {
	srv := ts.Server()

	wg := new(sync.WaitGroup)
	wg.Add(testCases)

	cli := ts.Client()

	cliErrs := make([]error, 0)
	cel := new(sync.Mutex)

	var done bool
	var started uint32
	var ended uint32

	i := 0
	go func() {
		for {
			select {
			case <-time.Tick(statDur):
				ss := srv.Stats()
				sent := cli.Sent()
				ngor := runtime.NumGoroutine()
				ncgo := runtime.NumCgoCall()

				log.Printf("i:%d started:%d c[sent:%d] s[%s] ended:%d errs:%d nGoR:%d nCGo:%d", i, started, sent, ss, ended, len(cliErrs), ngor, ncgo)

				if done {
					return
				}
			}
		}
	}()

	var sem *semaph.Semaph
	if maxDos > 0 {
		sem = semaph.NewSemaph(maxDos)
	}
	ctx := context.Background()

	for ; i < testCases; i++ {
		if sem != nil {
			sem.Acquire(ctx)
		}

		go func() {
			defer wg.Done()
			if sem != nil {
				defer sem.Release()
			}

			atomic.AddUint32(&started, 1)
			defer func() { atomic.AddUint32(&ended, 1) }()
			err := cli.Do()
			if err != nil {
				cel.Lock()
				cliErrs = append(cliErrs, err)
				cel.Unlock()
			}

		}()
	}

	wg.Wait()
	done = true

	log.Printf("errs:%d", len(cliErrs))

	errMsgs := make(map[string]int, 0)
	for _, e := range cliErrs {
		s := e.Error()
		_, ok := errMsgs[s]
		if !ok {
			errMsgs[s] = 0
		}

		errMsgs[s] += 1
	}

	for errm, c := range errMsgs {
		log.Printf("- err:%s", errm)
		log.Printf("  count:%d", c)
	}

	log.Printf("stats:[%s]", srv.Stats())
}
