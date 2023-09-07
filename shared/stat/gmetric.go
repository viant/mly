package stat

import (
	"time"

	"github.com/viant/gmetric"
	"github.com/viant/gmetric/counter"
)

type GMeter struct {
	*gmetric.Service

	Unit           time.Duration
	RecentDuration time.Duration
	NumRecent      int
}

func (g *GMeter) Op(location, name, desc string) *gmetric.Operation {
	return g.OperationCounter(location, name, desc, g.Unit, g.RecentDuration, g.NumRecent)
}

func (g *GMeter) MOp(location, name, desc string, p counter.Provider) *gmetric.Operation {
	return g.MultiOperationCounter(location, name, desc, g.Unit, g.RecentDuration, g.NumRecent, p)
}

func DefaultGMeter(m *gmetric.Service) *GMeter {
	return &GMeter{
		Service: m,

		Unit:           time.Microsecond,
		RecentDuration: time.Minute,
		NumRecent:      2,
	}
}
