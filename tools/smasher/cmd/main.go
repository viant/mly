package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type sleep struct {
	d     time.Duration
	f     int32
	c     int32
	cMax  int32
	cLock sync.Mutex
}

func (s *sleep) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	(&s.cLock).Lock()
	s.c += 1
	if s.c > s.cMax {
		s.cMax = s.c
	}
	(&s.cLock).Unlock()

	time.Sleep(s.d)

	atomic.AddInt32(&s.c, -1)
	atomic.AddInt32(&s.f, 1)
	w.WriteHeader(http.StatusAccepted)
}

func main() {
	s := new(http.Server)
	s.Addr = ":8889"
	sleeper := new(sleep)
	sleeper.d = 1 * time.Second
	s.Handler = sleeper

	var serverErr error
	go func() {
		serverErr = s.ListenAndServe()
		if serverErr != nil && serverErr != http.ErrServerClosed {
			log.Fatal(serverErr)
		}
	}()

	wg := new(sync.WaitGroup)
	testCases := 150000
	wg.Add(testCases)
	cli := new(http.Client)
	cliErrs := make([]error, 0)
	cel := new(sync.Mutex)

	var done bool
	var started int32

	i := 0
	go func() {
		for {
			select {
			case <-time.Tick(1 * time.Second):
				fmt.Printf("s.c:%d s.f:%d i:%d started:%d errs:%d\n", sleeper.c, sleeper.f, i, started, len(cliErrs))
				if done {
					return
				}
			}
		}
	}()

	for ; i < testCases; i++ {
		go func() {
			defer wg.Done()
			atomic.AddInt32(&started, 1)
			_, err := cli.Get("http://localhost:8889")
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
		fmt.Printf("err:%s c:%d\n", errm, c)
	}

	log.Printf("cMax:%d f:%d\n", sleeper.cMax, sleeper.f)

}
