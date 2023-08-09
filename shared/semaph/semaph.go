package semaph

import (
	"context"
	"fmt"
	"sync"
)

type Semaph struct {
	l   sync.Mutex // locks for modifying r
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

func (s *Semaph) R() int32 {
	return s.r
}

// Acquire will block if there are no more "tickets" left; otherwise will decrement number of tickets and continue.
// Caller must call Release() later, unless an error is returned, which should always be context.Context.Err().
func (s *Semaph) Acquire(ctx context.Context) error {
	s.l.Lock()

	c := make(chan bool, 1)

	for s.r <= 0 {
		canceled := false
		go func(cc *bool) {
			// this should unlock
			s.c.Wait()
			// this would lock
			if *cc {
				// outer routine would exist without unlocking
				defer s.l.Unlock()
				// "pass the torch" to next thing Wait()-ing
				s.c.Signal()
				return
			}
			c <- true
		}(&canceled)

		select {
		case <-ctx.Done():
			// this may still incur the Wait call completing
			canceled = true
			return ctx.Err()
		case <-c:
			// should've slept and locked
			// but we can wait again due to s.r == 0
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
		canceled := false
		go func(cc *bool) {
			s.c.Wait()
			if *cc {
				fmt.Printf("waited but canceled %s", f(s.r))
				defer s.l.Unlock()
				s.c.Signal()
				return
			}
			c <- true
		}(&canceled)

		select {
		case <-ctx.Done():
			fmt.Printf("canceled %s", f(s.r))
			canceled = true
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
