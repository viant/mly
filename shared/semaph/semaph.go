package semaph

import "sync"

type Semaph struct {
	l   sync.Mutex
	c   *sync.Cond
	i   int32
	max int32
}

func NewSemaph(max int32) *Semaph {
	s := new(Semaph)
	s.i = max
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

	for s.i <= 0 {
		// this should unlock
		s.c.Wait()
		// should've slept and locked
	}

	s.i -= 1
}

// Release will free up a "ticket", and if there were any waiting goroutines, will Signal() (one of) them.
func (s *Semaph) Release() {
	s.l.Lock()
	defer s.l.Unlock()

	if s.i == 0 {
		s.c.Signal()
	}

	if s.i < s.max {
		s.i += 1
	}
}
