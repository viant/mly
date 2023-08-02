// heavykeeper implementation
package hk

import (
	"github.com/migotom/heavykeeper"
	"github.com/viant/mly/shared/tracker"
)

type HK struct {
	tk *heavykeeper.TopK
}

func New(hk *heavykeeper.TopK) tracker.Tracker {
	return HK{tk: hk}
}

func (t HK) AddBytes(b []byte) {
	t.tk.AddBytes(b)
}

func (t HK) TopK() []tracker.Item {
	nodes := t.tk.List()
	items := make([]tracker.Item, len(nodes))

	for i, n := range nodes {
		items[i] = tracker.Item{
			Data:  n.Item,
			Count: n.Count,
		}
	}

	return items
}

func (t HK) Close() error {
	t.tk.Wait()
	return nil
}
