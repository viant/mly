package stat

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/viant/mly/shared/tracker/mg"
)

type TopK struct {
	mg  mg.MisraGries
	Top []E

	mod int
	c   uint32
	l   sync.RWMutex
}

type E struct {
	Error string
	Count uint64
}

type errorWrap struct {
	e error
}

func ErrorWrap(err error) errorWrap {
	return errorWrap{e: err}
}

// implements fmt.Stringer
func (e errorWrap) String() string {
	return e.e.Error()
}

// implements github.com/viant/gmetric/counter.CustomCounter
func (k *TopK) Aggregate(value interface{}) {
	b, ok := value.(fmt.Stringer)
	if !ok {
		fmt.Printf("%+v\n", value)
		return
	}

	s := b.String()

	k.l.RLock()
	k.mg.AddBytes([]byte(s))
	k.l.RUnlock()

	c := int(atomic.AddUint32(&k.c, 1))
	if k.mod <= 0 || c%k.mod == 0 {
		k.l.Lock()
		for i, it := range k.mg.TopK() {
			k.Top[i].Error = string(it.Data)
			k.Top[i].Count = it.Count
		}

		k.l.Unlock()
	}
}

func NewTopK(n, mod int) *TopK {
	mg := mg.NewK(n)

	return &TopK{
		mg:  mg,
		Top: make([]E, n, n),
		mod: mod,
	}
}
