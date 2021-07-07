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
	pool sync.Pool
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
	msg.mux.Unlock()

	return msg
}

func (p *messages) put(bs *Message) {
	if len(bs.buf) > bufferSize {
		return
	}
	p.pool.Put(bs)
}

//newMessages creates a new message pool
func newMessages(newDict func() *dictionary) Messages {
	keysLen := 0
	dict := newDict()
	if dict != nil {
		keysLen = len(dict.keys)
	}
	return &messages{
		pool: sync.Pool{
			New: func() interface{} {
				return &Message{
					buf:        make([]byte, bufferSize),
					keys:       make([]string, keysLen),
					rawKey:     make([]byte, 1024),
					dictionary: newDict(),
				}
			},
		},
	}
}
