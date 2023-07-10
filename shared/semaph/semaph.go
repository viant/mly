package semaph

import (
	"context"
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
func (s *Semaph) Acquire(ctx context.Context) error {
	s.l.Lock()

	c := make(chan bool, 1)
	for s.r <= 0 {
		go func() {
			// this should unlock
			s.c.Wait()
			c <- true
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c:
			// should've slept and locked
		}
	}

	defer s.l.Unlock()
	s.r -= 1
	return nil
}

func (s *Semaph) acquireDebug(ctx context.Context, f func(n int32) string) error {
	s.l.Lock()

	fmt.Printf("acquiring %s", f(s.r))

	c := make(chan bool, 1)
	for s.r <= 0 {
		go func() {
			s.c.Wait()
			c <- true
		}()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c:
			fmt.Printf("waited %s", f(s.r))
		}
	}

	defer s.l.Unlock()
	s.r -= 1
	return nil
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
