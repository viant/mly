package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/viant/mly/shared/common"
	"reflect"
	"strconv"
	"sync"
)

//Message represents a message
type (
	Message struct {
		batchSize int
		keyIndex  int
		buf       []byte
		index     int
		mux       sync.RWMutex
		pool      *messages
		keys      []string
		key       string
		keyLock   sync.Mutex
		buffer    *bytes.Buffer

		//used to represent vector
		multiKeys  [][]string
		multiKey   []string
		transient  []*transient
		cacheHits  []bool
		dictionary *Dictionary
	}
	transient struct {
		name   string
		values interface{}
		kind   reflect.Kind
	}
)

func (m *Message) Strings() []string {
	fields := m.dictionary.Fields()
	if len(m.transient) == 0 {
		return nil
	}
	var result = make([]string, 0)
	for i := 0; i < m.batchSize; i++ {
		record := map[string]interface{}{}

		for _, trans := range m.transient {
			field, ok := fields[trans.name]
			if !ok {
				continue
			}
			var values []string
			switch actual := trans.values.(type) {
			case []string:
				values = actual
			}
			value := values[0]
			if i < len(values) {
				value = values[i]
			}
			record[field.Name] = value

		}
		if data, _ := json.Marshal(record); len(data) > 0 {
			result = append(result, string(data))
		}
	}
	return result
}

func (m *Message) CacheHit(index int) bool {
	if index < len(m.cacheHits) {
		return m.cacheHits[index]
	}
	return false
}

func (m *Message) flushTransient(dim *transient, hasCacheHit bool) error {
	switch actual := dim.values.(type) {
	case []string:
		if hasCacheHit {
			var result = make([]string, m.requestBatchSize())
			j := 0
			for i, item := range actual {
				if m.CacheHit(i) {
					continue
				}
				result[j] = item
				j++
			}
			actual = result
		}
		m.stringsKey(dim.name, actual)
	case []int:
		if hasCacheHit {
			var result = make([]int, m.requestBatchSize())
			j := 0
			for i, item := range actual {
				if m.CacheHit(i) {
					continue
				}
				result[j] = item
				j++
			}
			actual = result
		}
		m.intsKey(dim.name, actual)
	case []float32:
		if hasCacheHit {
			var result = make([]float32, m.requestBatchSize())
			j := 0
			for i, item := range actual {
				if m.CacheHit(i) {
					continue
				}
				result[j] = item
				j++
			}
			actual = result
		}
		m.floatsKey(dim.name, actual)
	default:
		return fmt.Errorf("unsupported message type: %T", actual)
	}
	return nil

}

func (m *Message) SetBatchSize(batchSize int) {
	m.batchSize = batchSize
}

func (m *Message) BatchSize() int {
	return m.batchSize
}

//Size returns message size
func (m *Message) Size() int {
	return m.index
}

func (m *Message) start() {
	m.appendByte('{')
}

func (m *Message) end() error {
	if len(m.multiKeys) > 0 {
		if err := m.endInMultiKeyMode(); err != nil {
			return err
		}
		m.trim(',')
		m.appendString("}\n")
		return nil
	}

	if len(m.keys) > 0 {
		m.addCacheKey()
	}
	m.trim(',')
	m.appendString("}\n")
	return nil
}

//StringKey sets key/value pair
func (m *Message) StringKey(key, value string) {
	m.ensureMode("Strings")
	if key, index := m.dictionary.lookupString(key, value); index != unknownKeyField {
		m.keys[index] = key
	}
	m.appendByte('"')
	m.appendString(key)
	m.appendString(`":"`)
	m.appendString(value)
	m.appendString(`",`)
}

//ensureMode ensure that if multi keys are use no single message is allowed
func (m *Message) ensureMode(typeName string) {
	if m.batchSize > 0 {
		panic(fmt.Sprintf("use %vKey", typeName))
	}
}

//StringsKey sets key/values pair
func (m *Message) StringsKey(key string, values []string) {
	m.ensureMultiKeys(len(values))
	m.transient = append(m.transient, &transient{name: key, values: values, kind: reflect.String})
	var index int
	var keyValue string
	for i, value := range values {
		if len(m.multiKeys[i]) == 0 {
			m.multiKeys[i] = make([]string, m.dictionary.inputSize())
		}
		if keyValue, index = m.dictionary.lookupString(key, value); index != unknownKeyField {
			m.multiKeys[i][index] = keyValue
		}
	}
	if index <= 0 {
		return
	}

	m.expendKeysIfNeeded(len(values), index, keyValue)

}

// stringsKey sets key/values pair
func (m *Message) stringsKey(key string, values []string) {
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
	m.ensureMultiKeys(len(values))
	m.transient = append(m.transient, &transient{name: key, values: values, kind: reflect.Int64})

	var index int
	var intKeyValue int
	var keyValue string
	for i, value := range values {
		if len(m.multiKeys[i]) == 0 {
			m.multiKeys[i] = make([]string, m.dictionary.inputSize())
		}
		if intKeyValue, index = m.dictionary.lookupInt(key, value); index != unknownKeyField {
			keyValue = strconv.Itoa(intKeyValue)
			m.multiKeys[i][index] = keyValue
		}
	}
	m.expendKeysIfNeeded(len(values), index, keyValue)
}

func (m *Message) expendKeysIfNeeded(valuesLen int, index int, keyValue string) {
	if index < 0 {
		return
	}
	if valuesLen > 1 || m.batchSize <= 1 {
		return
	}
	for i := 1; i < m.batchSize; i++ {
		if len(m.multiKeys[i]) == 0 {
			m.multiKeys[i] = make([]string, m.dictionary.inputSize())
		}
		m.multiKeys[i][index] = keyValue
	}
}

//IntsKey sets key/values pair
func (m *Message) intsKey(key string, values []int) {
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
	m.ensureMode("Floats")
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
	m.ensureMultiKeys(len(values))
	m.transient = append(m.transient, &transient{name: key, values: values, kind: reflect.Float32})
	var index int
	var floatKeyValue float32
	var keyValue string
	for i, value := range values {
		if len(m.multiKeys[i]) == 0 {
			m.multiKeys[i] = make([]string, m.dictionary.inputSize())
		}
		if floatKeyValue, index = m.dictionary.lookupFloat(key, value); index != unknownKeyField {
			keyValue = strconv.FormatFloat(float64(floatKeyValue), 'f', 10, 32)
			m.multiKeys[i][index] = keyValue
		}
	}
	m.expendKeysIfNeeded(len(values), index, keyValue)
}

//FloatsKey sets key/values pair
func (m *Message) floatsKey(key string, values []float32) {
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

// trim removes the last character from the buffer
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

//Release releases message to the grpcPool
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

func (m *Message) FlagCacheHit(index int) {
	diff := (index - len(m.cacheHits)) + 1
	if diff > 0 {
		m.cacheHits = append(m.cacheHits, make([]bool, diff)...)
	}
	m.cacheHits[index] = true
}

//CacheKeyAt returns cache key for supplied index
func (m *Message) CacheKeyAt(index int) string {
	m.keyLock.Lock()
	defer m.keyLock.Unlock()
	if m.batchSize == 0 {
		return m.CacheKey()
	}
	if len(m.multiKey) == 0 {
		m.multiKey = make([]string, len(m.multiKeys))
	}
	if m.multiKey[index] != "" {
		return m.multiKey[index]
	}
	m.multiKey[index] = buildKey(m.multiKeys[index], m.buffer)
	m.buffer.Reset()
	return m.multiKey[index]
}

//CacheKey returns cache key
func (m *Message) CacheKey() string {
	if m.key != "" || len(m.keys) == 0 {
		return m.key
	}
	m.key = buildKey(m.keys, m.buffer)
	return m.key
}

func buildKey(keys []string, buffer *bytes.Buffer) string {
	buffer.WriteString(keys[0])

	//
	for i := 1; i < len(keys); i++ {
		buffer.WriteByte(common.KeyDelimiter)
		buffer.WriteString(keys[i])
	}
	rawKey := buffer.Bytes()
	return string(rawKey)
}

func (m *Message) addCacheKey() {
	aKey := m.CacheKey()
	if aKey == "" {
		return
	}
	m.StringKey(common.CacheKey, aKey)
}

func (m *Message) endInMultiKeyMode() error {
	hasCacheHit := m.hasCacheHit()

	m.IntKey(common.BatchSizeKey, m.requestBatchSize())
	for _, item := range m.transient {
		if err := m.flushTransient(item, hasCacheHit); err != nil {
			return err
		}
	}

	var multiKey []string
	if len(m.multiKeys) > 0 {
		for i := range m.multiKeys { //
			if m.CacheHit(i) {
				continue
			}
			multiKey = append(multiKey, m.CacheKeyAt(i))
		}
		m.stringsKey(common.CacheKey, multiKey)
	}
	return nil
}

func (m *Message) requestBatchSize() int {
	cacheHits := 0
	for _, hit := range m.cacheHits {
		if hit {
			cacheHits++
		}
	}
	return m.batchSize - cacheHits
}

func (m *Message) ensureMultiKeys(l int) {
	if m.batchSize == 0 {
		m.batchSize = l
	}
	if len(m.multiKeys) == 0 {
		m.multiKeys = make([][]string, m.batchSize)
	}
}

func (m *Message) hasCacheHit() bool {
	if len(m.cacheHits) == 0 {
		return false
	}
	for _, hit := range m.cacheHits {
		if hit {
			return true
		}
	}
	return false
}
