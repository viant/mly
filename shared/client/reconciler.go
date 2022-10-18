package client

import (
	"fmt"
	"github.com/viant/xunsafe"
	"reflect"
	"unsafe"
)

//ReconcileData reconciles response with locally cached retrieved data
func ReconcileData(debug bool, target interface{}, cachable Cachable, cached []interface{}) error {
	targetType := reflect.TypeOf(target).Elem()
	dataPtr := xunsafe.AsPointer(target)

	if debug {
		fmt.Printf("reconciling: %T %+v\n", target, target)
	}
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

	prevSlice := (*reflect.SliceHeader)(dataPtr)
	newData := reflect.MakeSlice(xSlice.Type, batchSize, batchSize)
	newDataPtr := xunsafe.ValuePointer(&newData)
	hadDataOnlyInCache := len(cached) > 0
	for i, cacheEntry := range cached {
		if cacheEntry == nil {
			hadDataOnlyInCache = false
		}
		itemPtrAddr := xSlice.PointerAt(newDataPtr, uintptr(i))
		*(*unsafe.Pointer)(itemPtrAddr) = xunsafe.AsPointer(cached[i])
		if debug {
			fmt.Printf("updated from cache[%v] %+v\n", i, cacheEntry)
		}

	}
	if hadDataOnlyInCache {
		return nil
	}

	offsets := buildOffsets(batchSize, cachable)
	if debug {
		fmt.Printf("offsets map: %+v\n", offsets)
	}
	for index := 0; index < prevSlice.Len; index++ {
		value := xSlice.ValuePointerAt(dataPtr, index)
		cacheableIndex := offsets[index]
		if cachable.CacheHit(cacheableIndex) {
			continue
		}
		if debug {
			fmt.Printf("updated from response[%v->%v]: %+v\n", index, cacheableIndex, value)
		}
		itemPtrAddr := xSlice.PointerAt(newDataPtr, uintptr(cacheableIndex))
		*(*unsafe.Pointer)(itemPtrAddr) = xunsafe.AsPointer(value)
	}

	nextSlice := *(*reflect.SliceHeader)(newDataPtr)
	prevSlice.Cap = nextSlice.Cap
	prevSlice.Len = nextSlice.Len
	prevSlice.Data = nextSlice.Data
	return nil
}

func buildOffsets(batchSize int, cachable Cachable) []int {
	var offsets = make([]int, batchSize) //index offsets to recncile local cache hits

	offset := 0
	for i := 0; i < batchSize; i++ {
		offsets[offset] = i
		if !cachable.CacheHit(i) {
			offset++
		}
	}
	return offsets
}
