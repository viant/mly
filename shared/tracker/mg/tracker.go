// Misra-Gries '84 "heavy hitters" for frequent elements.
package mg

import (
	"sync"

	"github.com/cyningsun/heavy-hitters/misragries"
	"github.com/viant/mly/shared/tracker"
)

type MisraGries struct {
	mg *misragries.MisraGries
	l  *sync.Mutex
}

func New(mg *misragries.MisraGries) tracker.Tracker {
	return MisraGries{mg: mg, l: new(sync.Mutex)}
}

func (m MisraGries) AddBytes(b []byte) {
	m.l.Lock()
	defer m.l.Unlock()
	m.mg.ProcessElement(string(b))
}

func (m MisraGries) TopK() []tracker.Item {
	counters := m.mg.TopK()
	items := make([]tracker.Item, len(counters))
	i := 0
	for k, c := range counters {
		items[i] = tracker.Item{
			Data: []byte(k),
			// this should be safe since if c <= 0 it should be removed
			Count: uint64(c),
		}
		i++
	}

	return items
}

func (m MisraGries) Close() error {
	return nil
}
