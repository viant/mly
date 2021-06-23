package buffer

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
	"net/http/httputil"
)

//Read reads data with buffer Pool
func Read(pool httputil.BufferPool, reader io.Reader) ([]byte, int, error) {
	data := pool.Get()
	readTotal := 0
	offset := 0
	for i := 0; i < len(data); i++ {
		if offset >= len(data) {
			pool.Put(data)
			return nil, 0, errors.Errorf("buffer too small: %v", len(data))
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

type Reader struct {
	Data []byte
	io.Reader
}

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
