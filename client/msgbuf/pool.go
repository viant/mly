package msgbuf

import (
	"sync"
)

const bufferSize = 8 * 1024

var _pool = newPool()

type pool struct {
	pool sync.Pool
}

//Borrow returns message
func Borrow() *Message {
	return _pool.Borrow()
}

func (p *pool) Borrow() *Message {
	buffer := p.pool.Get().(*Message)
	buffer.index = 0
	if len(buffer.buf) > bufferSize {
		buffer.buf = buffer.buf[:bufferSize]
	}
	buffer.pool = p
	return buffer
}

func (p *pool) put(bs *Message) {
	if len(bs.buf) > bufferSize {
		return
	}
	p.pool.Put(bs)
}

func newPool() *pool {
	return &pool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Message{buf: make([]byte, bufferSize)}
			},
		},
	}
}
