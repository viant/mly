package buffer

import "sync/atomic"

//Pool represents data pool
type Pool struct {
	channel     chan []byte
	poolMaxSize int32
	count       int32
	bufferSize  int
}

//Get returns bytes
func (p *Pool) Get() (result []byte) {
	select {
	case result = <-p.channel:
	default:
		result = make([]byte, p.bufferSize)
	}
	atomic.AddInt32(&p.count, -1)
	return result
}

//Put put data back to the pool
func (p *Pool) Put(b []byte) {
	if len(b) != p.bufferSize {
		return
	}
	if atomic.AddInt32(&p.count, 1) <= p.poolMaxSize {
		select {
		case p.channel <- b:
		default: //If the Pool is full, discard the buffer.
		}
	}
}

//New creates a httputil.BufferPool Pool.
func New(poolMaxSize, bufferSize int) *Pool {
	result := &Pool{
		poolMaxSize: int32(poolMaxSize),
		channel:     make(chan []byte, poolMaxSize),
		bufferSize:  bufferSize,
	}
	for i := 0; i < poolMaxSize; i++ {
		result.Put(make([]byte, bufferSize))
	}
	return result
}
