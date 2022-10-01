package client

import (
	"fmt"
	"github.com/viant/xunsafe"
	"reflect"
	"unsafe"
)

//ReconcileData reconciles response with locally cached retrieved data
func ReconcileData(target interface{}, cachable Cachable, cached []interface{}) error {
	targetType := reflect.TypeOf(target).Elem()
	dataPtr := xunsafe.AsPointer(target)

	switch targetType.Kind() {
	case reflect.Struct:
		if !cachable.CacheHit(0) {
			return nil
		}
		*(*unsafe.Pointer)(dataPtr) = *(*unsafe.Pointer)(xunsafe.AsPointer(cached[0]))
		return nil
	case reflect.Slice:
	default:
		return fmt.Errorf("unsupported target type expected *T or []*T, but had: %T", target)
	}
	xSlice := xunsafe.NewSlice(targetType.Elem())
	batchSize := cachable.BatchSize()
	var offsets = make([]int, batchSize) //index offsets to recncile local cache hits
	offset := 0
	for i := 0; i < batchSize; i++ {
		offsets[offset] = i
		if !cachable.CacheHit(i) {
			offset++
		}
	}
	newData := reflect.MakeSlice(xSlice.Type, batchSize, batchSize)
	newDataPtr := xunsafe.ValuePointer(&newData)

	for index := 0; index < batchSize; index++ {
		value := xSlice.ValuePointerAt(dataPtr, index)
		cacheableIndex := offsets[index]
		if cachable.CacheHit(cacheableIndex) {
			continue
		}
		itemPtrAddr := xSlice.PointerAt(newDataPtr, uintptr(cacheableIndex))
		*(*unsafe.Pointer)(itemPtrAddr) = xunsafe.AsPointer(value)
	}

	for i, cacheEntry := range cached {
		if cacheEntry == nil {
			continue
		}
		itemPtrAddr := xSlice.PointerAt(newDataPtr, uintptr(i))
		*(*unsafe.Pointer)(itemPtrAddr) = xunsafe.AsPointer(cached[i])
	}

	nextSlice := *(*reflect.SliceHeader)(newDataPtr)
	prevSlice := (*reflect.SliceHeader)(dataPtr)
	prevSlice.Cap = nextSlice.Cap
	prevSlice.Len = nextSlice.Len
	prevSlice.Data = nextSlice.Data
	return nil
}
