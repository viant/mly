package semaph

import (
	"fmt"
	"sync"
)

type Semaph struct {
	l   sync.Mutex
	c   *sync.Cond
	r   int32 // i.e. remaining tickets
	max int32
}

func NewSemaph(max int32) *Semaph {
	s := new(Semaph)
	s.r = max
	s.max = max

	s.l = sync.Mutex{}
	s.c = sync.NewCond(&s.l)

	return s
}

// Acquire will block if there are no more "tickets" left; otherwise will decrement number of tickets and continue.
// Caller must call Release() later.
// TODO How to incorporate Context?
func (s *Semaph) Acquire() {
	s.l.Lock()
	defer s.l.Unlock()

	for s.r <= 0 {
		// this should unlock
		s.c.Wait()
		// should've slept and locked
	}

	s.r -= 1
}

func (s *Semaph) acquireDebug(f func(n int32) string) {
	s.l.Lock()
	defer s.l.Unlock()

	fmt.Printf("acquiring %s", f(s.r))

	for s.r <= 0 {
		// this should unlock
		s.c.Wait()
		// should've slept and locked
		fmt.Printf("waited %s", f(s.r))
	}

	s.r -= 1
}

// Release will free up a "ticket", and if there were any waiting goroutines, will Signal() (one of) them.
func (s *Semaph) Release() {
	s.l.Lock()

	if s.r < s.max {
		s.r += 1
		s.c.Signal()
	}

	s.l.Unlock()
}

func (s *Semaph) releaseDebug(f func(n int32) string) {
	s.l.Lock()

	fmt.Printf("%s", f(s.r))

	if s.r < s.max {
		s.r += 1
		s.c.Signal()
	}

	s.l.Unlock()
}
