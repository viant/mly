package smasher

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type H interface {
	Do()
}

type Lambda struct {
	H    H
	F    uint64
	A    uint64
	AMax uint64

	cLock sync.Mutex
}

func (s *Lambda) Stats() string {
	return fmt.Sprintf("active:%d activeMax:%d fin:%d", s.A, s.AMax, s.F)
}

type Sleeper struct {
	D time.Duration
}

func (s *Sleeper) Do() {
	time.Sleep(s.D)
}

func (s *Lambda) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	(&s.cLock).Lock()
	s.A += 1
	if s.A > s.AMax {
		s.AMax = s.A
	}
	(&s.cLock).Unlock()

	s.H.Do()

	atomic.AddUint64(&s.A, ^uint64(0))
	atomic.AddUint64(&s.F, 1)

	w.WriteHeader(http.StatusAccepted)
}

type NoOp struct{}

func (a *NoOp) Active() uint64 {
	return 0
}

func (a *NoOp) MaxActive() uint64 {
	return 0
}

func (a *NoOp) Finished() uint64 {
	return 0
}
