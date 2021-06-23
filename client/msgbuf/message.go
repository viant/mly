package msgbuf

import (
	"io"
	"strconv"
	"sync"
)

//Message represents a message
type Message struct {
	buf   []byte
	index int
	mux   sync.Mutex
	pool  *pool
}

func (m *Message) Size() int {
	return m.index
}

func (m *Message) StartObject(key string) {
	m.nextItemIfNeeded()
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":`)
	m.appendByte('{')
}

func (m *Message) Start() {
	m.nextItemIfNeeded()
	m.appendByte('{')
}

func (m *Message) nextItemIfNeeded() {
	if m.index == 0 {
		return
	}
	switch m.buf[m.index-1] {
	case ',', '{', '[':
	default:
		m.appendByte(',')
	}
}

func (m *Message) End() {
	m.trim(',')
	m.appendString("}")
}

func (m *Message) EndObject() {
	m.trim(',')
	m.appendByte('}')
}

func (m *Message) StartArray(key string) {
	m.nextItemIfNeeded()
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":[`)
}

func (m *Message) EndArray() {
	m.trim(',')
	m.appendByte(']')
}

func (m *Message) StringKey(key, value string) {
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":"`)
	m.appendString(value)
	m.appendString(`",`)
}

func (m *Message) StringsKey(key string, values []string) {
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":[`)
	for i, item := range values {
		if i > 0 {
			m.appendByte(',')
		}
		m.appendByte('"')
		m.appendString(item)
		m.appendByte('"')
	}
	m.appendString(`],`)
}

func (m *Message) IntsKey(key string, values []int) {
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":[`)
	for i, item := range values {
		if i > 0 {
			m.appendByte(',')
		}
		m.appendString(strconv.Itoa(item))
	}
	m.appendString(`],`)
}

func (m *Message) IntKey(key string, value int) {
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":`)
	m.appendString(strconv.Itoa(value))
	m.appendString(`,`)
}

func (m *Message) FloatKey(key string, value float64) {
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":`)
	m.appendFloat(value, 64)
	m.appendString(`,`)
}

func (m *Message) FloatsKey(key string, values []float64) {
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":`)
	for i, item := range values {
		if i > 0 {
			m.appendByte(',')
		}
		m.appendFloat(item, 64)
	}
	m.appendString(`,`)
}

func (m *Message) BoolKey(key string, value bool) {
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":`)
	if value {
		m.appendString("true")
	} else {
		m.appendString("false")

	}
	m.appendString(`,`)
}

func (m *Message) WriteTo(w io.Writer) (int64, error) {
	if m.index == 0 {
		return 0, nil
	}
	offset := 0
	for {
		n, err := w.Write(m.buf[offset:m.index])
		offset += n
		if err != nil || offset >= m.index {
			return int64(offset), err
		}
	}
}

func (m *Message) quotedString(s string) {
	m.appendByte('"')
	m.appendString(s)
	m.appendByte('"')
}

func (m *Message) appendBytes(bs []byte) {
	bsLen := len(bs)
	if bsLen == 0 {
		return
	}
	if bsLen+m.index >= len(m.buf) {
		size := bufferSize
		if size < bsLen {
			size = bsLen
		}
		m.buf = append(m.buf, make([]byte, size)...)
	}
	copy(m.buf[m.index:], bs)
	m.index += bsLen
}

func (m *Message) appendByte(bs byte) {
	if m.index+1 >= len(m.buf) {
		newBuffer := make([]byte, bufferSize)
		m.buf = append(m.buf, newBuffer...)
	}
	m.buf[m.index] = bs
	m.index++
}

func (m *Message) appendString(s string) {
	sLen := len(s)
	if sLen == 0 {
		return
	}
	if sLen+m.index >= len(m.buf) {
		size := bufferSize
		if size < sLen {
			size = sLen
		}
		newBuffer := make([]byte, size)
		m.buf = append(m.buf, newBuffer...)
	}
	copy(m.buf[m.index:], s)
	m.index += sLen
}

func (m *Message) Bytes() []byte {
	return m.buf[:m.index]
}

// appendInt appends an integer to the underlying buffer (assuming base 10).
func (m *Message) appendInt(i int64) {
	s := strconv.FormatInt(i, 10)
	m.appendString(s)
}

// appendUint appends an unsigned integer to the underlying buffer (assuming
// base 10).
func (m *Message) appendUint(i uint64) {
	s := strconv.FormatUint(i, 10)
	m.appendString(s)
}

// appendBool appends a bool to the underlying buffer.
func (m *Message) appendBool(v bool) {
	s := strconv.FormatBool(v)
	m.appendString(s)
}

// AppendFloat appends a float to the underlying buffer.
func (m *Message) appendFloat(f float64, bitSize int) {
	s := strconv.FormatFloat(f, 'f', -1, bitSize)
	m.appendString(s)
}

// trim trims any final character from the buffer
func (m *Message) trim(ch byte) {
	if m.buf[m.index-1] == ch && m.index > 0 {
		m.index--
	}
}

func (m *Message) Release() {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.pool == nil {
		return
	}
	pool := m.pool
	m.pool = nil
	pool.put(m)
}
