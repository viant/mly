package client

import (
	"github.com/viant/mly/shared/common"
	"strconv"
	"sync"
)

//Message represents a message
type Message struct {
	buf        []byte
	index      int
	mux        sync.RWMutex
	pool       *messages
	keys       []string
	key        string
	rawKey     []byte
	dictionary *dictionary
}

//Size returns message size
func (m *Message) Size() int {
	return m.index
}

func (m *Message) start() {
	m.appendByte('{')
}

func (m *Message) end() {
	if len(m.keys) > 0 {
		m.addCacheKey()
	}
	m.trim(',')
	m.appendString("}\n")
}

//StringKey sets key/value pair
func (m *Message) StringKey(key, value string) {
	if key, index := m.dictionary.lookupString(key, value); index != unknownKeyField {
		m.keys[index] = key
	}
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":"`)
	m.appendString(value)
	m.appendString(`",`)
}

//StringsKey sets key/values pair
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

//IntsKey sets key/values pair
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

//IntKey sets key/value pair
func (m *Message) IntKey(key string, value int) {
	if key, index := m.dictionary.lookupInt(key, value); index != unknownKeyField {
		m.keys[index] = strconv.Itoa(key)
	}
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":`)
	m.appendString(strconv.Itoa(value))
	m.appendString(`,`)
}

//FloatKey sets key/value pair
func (m *Message) FloatKey(key string, value float32) {
	if key, index := m.dictionary.lookupFloat(key, value); index != unknownKeyField {
		m.keys[index] = strconv.FormatFloat(float64(key), 'f', 10, 32)
	}
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":`)
	m.appendFloat(value, 32)
	m.appendString(`,`)
}

//FloatsKey sets key/values pair
func (m *Message) FloatsKey(key string, values []float32) {
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":`)
	for i, item := range values {
		if i > 0 {
			m.appendByte(',')
		}
		m.appendFloat(item, 32)
	}
	m.appendString(`,`)
}

//BoolKey sets key/value pair
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

//Bytes returns message bytes
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
func (m *Message) appendFloat(f float32, bitSize int) {
	s := strconv.FormatFloat(float64(f), 'f', -1, bitSize)
	m.appendString(s)
}

// trim trims any final character from the buffer
func (m *Message) trim(ch byte) {
	if m.buf[m.index-1] == ch && m.index > 0 {
		m.index--
	}
}

func (m *Message) isValid() bool {
	m.mux.RLock()
	pool := m.pool
	m.mux.RUnlock()
	return pool != nil
}

//Release releases message to the pool
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

//CacheKey returns cache key
func (m *Message) CacheKey() string {
	if m.key != "" {
		return m.key
	}
	offset := copy(m.rawKey, m.keys[0])
	for i := 1; i < len(m.keys); i++ {
		m.rawKey[offset] = common.KeyDelimiter
		offset++
		copied := copy(m.rawKey[offset:], m.keys[i])
		offset += copied
	}
	rawKey := m.rawKey[:offset]
	m.key = string(rawKey)
	return m.key
}

func (m *Message) addCacheKey() {
	aKey := m.CacheKey()
	if aKey == "" {
		return
	}
	m.StringKey("_key", aKey)
}
