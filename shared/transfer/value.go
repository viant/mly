package transfer

import (
	"fmt"
	"reflect"

	"github.com/francoispqt/gojay"
	"github.com/viant/toolbox"
)

type (
	Value interface {
		Set(values ...interface{}) error
		Key() string
		ValueAt(index int) interface{}
		UnmarshalJSONArray(dec *gojay.Decoder) error
		Len() int
		// Will make a new []interface{} that copies old value and fills the rest with the first element.
		// Panics if batchSize is less than current Values size.
		Feed(batchSize int) interface{}
	}

	Values []Value

	Strings struct {
		Name   string
		Values []string
	}
	Int32s struct {
		Name   string
		Values []int32
	}
	Int64s struct {
		Name   string
		Values []int64
	}
	Bools struct {
		Name   string
		Values []bool
	}
	Float32s struct {
		Name   string
		Values []float32
	}
	Float64s struct {
		Name   string
		Values []float64
	}
)

func (s *Strings) ValueAt(index int) interface{} {
	if index >= len(s.Values) {
		return s.Values[0]
	}
	return s.Values[index]
}

func (s *Strings) Feed(batchSize int) interface{} {
	var result = make([][]string, batchSize)
	for i, item := range s.Values {
		result[i] = []string{item}
	}

	for i := len(s.Values); i < batchSize; i++ {
		result[i] = []string{s.Values[0]}
		s.Values = append(s.Values, s.Values[0])
	}
	return result
}

func (s *Strings) Key() string {
	return s.Name
}

func (s *Strings) Len() int {
	return len(s.Values)
}

func (s *Strings) Set(values ...interface{}) error {
	s.Values = make([]string, len(values))
	for i, v := range values {
		s.Values[i] = toolbox.AsString(v)
	}
	return nil
}

// UnmarshalJSONArray decodes JSON array elements into slice
func (a *Strings) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value string
	if err := dec.String(&value); err != nil {
		return err
	}
	a.Values = append(a.Values, value)
	return nil
}

// MarshalJSONArray encodes arrays into JSON
func (a Strings) MarshalJSONArray(enc *gojay.Encoder) {
	for i := 0; i < len(a.Values); i++ {
		enc.String(a.Values[i])
	}
}

// IsNil checks if array is nil
func (v Strings) IsNil() bool {
	return len(v.Values) == 0
}

func (s *Int32s) Feed(batchSize int) interface{} {
	var result = make([][]int32, batchSize)
	for i, item := range s.Values {
		result[i] = []int32{item}
	}
	for i := len(s.Values); i < batchSize; i++ {
		result[i] = []int32{s.Values[0]}
		s.Values = append(s.Values, s.Values[0])
	}
	return result
}

func (s *Int32s) ValueAt(index int) interface{} {
	if index >= len(s.Values) {
		return s.Values[0]
	}
	return s.Values[index]
}

func (s *Int32s) Key() string {
	return s.Name
}

func (s *Int32s) Len() int {
	return len(s.Values)
}

func (s *Int32s) Set(values ...interface{}) error {
	s.Values = make([]int32, len(values))
	for i, v := range values {
		val, err := toolbox.ToInt(v)
		if err != nil {
			return err
		}
		s.Values[i] = int32(val)
	}
	return nil
}

// UnmarshalJSONArray decodes JSON array elements into slice
func (a *Int32s) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value int
	if err := dec.Int(&value); err != nil {
		return err
	}
	a.Values = append(a.Values, int32(value))
	return nil
}

// MarshalJSONArray encodes arrays into JSON
func (a Int32s) MarshalJSONArray(enc *gojay.Encoder) {
	for i := 0; i < len(a.Values); i++ {
		enc.Int(int(a.Values[i]))
	}
}

// IsNil checks if array is nil
func (v Int32s) IsNil() bool {
	return len(v.Values) == 0
}

func (s *Int64s) Feed(batchSize int) interface{} {
	var result = make([][]int64, batchSize)
	for i, item := range s.Values {
		result[i] = []int64{item}
	}
	for i := len(s.Values); i < batchSize; i++ {
		result[i] = []int64{s.Values[0]}
		s.Values = append(s.Values, s.Values[0])
	}
	return result
}

func (s *Int64s) Len() int {
	return len(s.Values)
}

func (s *Int64s) ValueAt(index int) interface{} {
	if index >= len(s.Values) {
		return s.Values[0]
	}
	return s.Values[index]
}

func (s *Int64s) Key() string {
	return s.Name
}

func (v *Int64s) Set(values ...interface{}) error {
	v.Values = make([]int64, len(values))
	for i, item := range values {
		val, err := toolbox.ToInt(item)
		if err != nil {
			return err
		}
		v.Values[i] = int64(val)
	}
	return nil
}

// UnmarshalJSONArray decodes JSON array elements into slice
func (v *Int64s) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value int
	if err := dec.Int(&value); err != nil {
		return err
	}
	v.Values = append(v.Values, int64(value))
	return nil
}

// MarshalJSONArray encodes arrays into JSON
func (v Int64s) MarshalJSONArray(enc *gojay.Encoder) {
	for i := 0; i < len(v.Values); i++ {
		enc.Int(int(v.Values[i]))
	}
}

// IsNil checks if array is nil
func (v Int64s) IsNil() bool {
	return len(v.Values) == 0
}

func (s *Bools) Feed(batchSize int) interface{} {
	var result = make([][]bool, batchSize)
	for i, item := range s.Values {
		result[i] = []bool{item}
	}
	for i := len(s.Values); i < batchSize; i++ {
		result[i] = []bool{s.Values[0]}
		s.Values = append(s.Values, s.Values[0])
	}
	return result
}

func (s *Bools) Len() int {
	return len(s.Values)
}

func (s *Bools) ValueAt(index int) interface{} {
	if index >= len(s.Values) {
		return s.Values[0]
	}
	return s.Values[index]
}

func (s *Bools) Key() string {
	return s.Name
}

func (s *Bools) Set(values ...interface{}) error {
	s.Values = make([]bool, len(values))
	for i, v := range values {
		val, err := toolbox.ToBoolean(v)
		if err != nil {
			return err
		}
		s.Values[i] = val
	}
	return nil
}

// UnmarshalJSONArray decodes JSON array elements into slice
func (v *Bools) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value bool
	if err := dec.Bool(&value); err != nil {
		return err
	}
	v.Values = append(v.Values, value)
	return nil
}

// MarshalJSONArray encodes arrays into JSON
func (v Bools) MarshalJSONArray(enc *gojay.Encoder) {
	for i := 0; i < len(v.Values); i++ {
		enc.Bool(v.Values[i])
	}
}

// IsNil checks if array is nil
func (v Bools) IsNil() bool {
	return len(v.Values) == 0
}

func (s *Float32s) Feed(batchSize int) interface{} {
	var result = make([][]float32, batchSize)
	for i, item := range s.Values {
		result[i] = []float32{item}
	}
	for i := len(s.Values); i < batchSize; i++ {
		result[i] = []float32{s.Values[0]}
	}
	return result
}

func (s *Float32s) Len() int {
	return len(s.Values)
}

func (s *Float32s) ValueAt(index int) interface{} {
	if index >= len(s.Values) {
		return s.Values[0]
	}
	return s.Values[index]
}

func (s *Float32s) Key() string {
	return s.Name
}

func (s *Float32s) Set(values ...interface{}) error {
	s.Values = make([]float32, len(values))
	for i, v := range values {
		val, err := toolbox.ToFloat(v)
		if err != nil {
			return err
		}
		s.Values[i] = float32(val)
	}
	return nil
}

// UnmarshalJSONArray decodes JSON array elements into slice
func (v *Float32s) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value float32
	if err := dec.Float32(&value); err != nil {
		return err
	}
	v.Values = append(v.Values, value)
	return nil
}

// MarshalJSONArray encodes arrays into JSON
func (v Float32s) MarshalJSONArray(enc *gojay.Encoder) {
	for i := 0; i < len(v.Values); i++ {
		enc.Float32(v.Values[i])
	}
}

// IsNil checks if array is nil
func (v Float32s) IsNil() bool {
	return len(v.Values) == 0
}

func (s *Float64s) Feed(batchSize int) interface{} {
	var result = make([][]float64, batchSize)
	for i, item := range s.Values {
		result[i] = []float64{item}
	}
	for i := len(s.Values); i < batchSize; i++ {
		result[i] = []float64{s.Values[0]}
		s.Values = append(s.Values, s.Values[0])
	}
	return result
}

func (s *Float64s) Len() int {
	return len(s.Values)
}

func (s *Float64s) ValueAt(index int) interface{} {
	return s.Values[index]
}

func (s *Float64s) Key() string {
	return s.Name
}

func (s *Float64s) Set(values ...interface{}) error {
	s.Values = make([]float64, len(values))
	for i, v := range values {
		val, err := toolbox.ToFloat(v)
		if err != nil {
			return err
		}
		s.Values[i] = val
	}
	return nil
}

// UnmarshalJSONArray decodes JSON array elements into slice
func (v *Float64s) UnmarshalJSONArray(dec *gojay.Decoder) error {
	var value float64
	if err := dec.Float64(&value); err != nil {
		return err
	}
	v.Values = append(v.Values, value)
	return nil
}

// MarshalJSONArray encodes arrays into JSON
func (v Float64s) MarshalJSONArray(enc *gojay.Encoder) {
	for i := 0; i < len(v.Values); i++ {
		enc.Float64(v.Values[i])
	}
}

// IsNil checks if array is nil
func (v Float64s) IsNil() bool {
	return len(v.Values) == 0
}

func (v *Values) ValueAt(index int) Value {
	if index < len(*v) {
		return (*v)[index]
	}
	*v = append(*v, make([]Value, 1+index-len(*v))...)
	return (*v)[index]
}

// SetAt will create []Value at index or extend current array to support index
func (v *Values) SetAt(index int, name string, kind reflect.Kind) (Value, error) {
	if index >= len(*v) {
		*v = append(*v, make([]Value, 1+index-len(*v))...)
	}

	err := v.allocate(index, name, kind)
	if err != nil {
		return nil, err
	}

	return (*v)[index], nil
}

func (v *Values) allocate(index int, name string, kind reflect.Kind) error {
	switch kind {
	case reflect.Bool:
		(*v)[index] = &Bools{Name: name}
	case reflect.Int32:
		(*v)[index] = &Int32s{Name: name}
	case reflect.Int64, reflect.Int, reflect.Uint, reflect.Uint64:
		(*v)[index] = &Int64s{Name: name}
	case reflect.Float64:
		(*v)[index] = &Float64s{Name: name}
	case reflect.Float32:
		(*v)[index] = &Float32s{Name: name}
	case reflect.String:
		(*v)[index] = &Strings{Name: name}
	default:
		return fmt.Errorf("unsupported kind %v:%v", name, kind.String())
	}
	return nil
}
