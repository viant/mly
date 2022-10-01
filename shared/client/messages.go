package client

import (
	"sync"
)

const bufferSize = 8 * 1024

//Messages represent a message
type Messages interface {
	Borrow() *Message
}

type messages struct {
	pool    sync.Pool
	nweDict func() *Dictionary
}

func (p *messages) Borrow() *Message {
	msg := p.pool.Get().(*Message)
	msg.index = 0
	if len(msg.buf) > bufferSize {
		msg.buf = msg.buf[:bufferSize]
	}
	for i := range msg.keys {
		msg.keys[i] = ""
	}
	msg.mux.Lock()
	msg.pool = p
	msg.key = ""
	msg.buffer.Reset()
	msg.multiKey = nil
	msg.multiKeys = nil
	msg.transient = nil
	msg.cacheHits = nil
	msg.dictionary = p.nweDict()
	msg.mux.Unlock()
	return msg
}

func (p *messages) put(bs *Message) {
	if len(bs.buf) > bufferSize {
		return
	}
	p.pool.Put(bs)
}

//NewMessages creates a new message grpcPool
func NewMessages(newDict func() *Dictionary) Messages {
	keysLen := 0
	dict := newDict()
	if dict != nil {
		keysLen = dict.KeysLen()
	}
	return &messages{
		nweDict: newDict,
		pool: sync.Pool{
			New: func() interface{} {
				return &Message{
					buf:        make([]byte, bufferSize),
					keys:       make([]string, keysLen),
					dictionary: newDict(),
				}
			},
		},
	}
}
