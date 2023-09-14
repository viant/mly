package buffer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http/httputil"
)

var ErrBufferTooSmall error = errors.New("entity too large - buffer too small")

// Read reads data with buffer Pool
func Read(pool httputil.BufferPool, reader io.Reader) ([]byte, int, error) {
	data := pool.Get()
	readTotal := 0
	offset := 0
	for i := 0; i < len(data); i++ {
		if offset >= len(data) {
			pool.Put(data)
			return nil, 0, fmt.Errorf("%w - buffer size:%d, tried to read:%d", ErrBufferTooSmall, len(data), offset)
		}
		read, err := reader.Read(data[offset:])
		offset += read
		readTotal += read
		if err != nil {
			if err == io.EOF {
				break
			}
			pool.Put(data)
			return nil, 0, err
		}
		if read == 0 {
			break
		}
	}
	return data, readTotal, nil
}

//Reader represents a reader
type Reader struct {
	Data []byte
	io.Reader
}

//Close closes reader
func (r *Reader) Close() error {
	return nil
}

//NewReader creates a new reader closer wrapper
func NewReader(data []byte) io.ReadCloser {
	return &Reader{
		Data:   data,
		Reader: bytes.NewReader(data),
	}
}
