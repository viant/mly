package client

import (
	"fmt"
	"log"
	"reflect"
	"unsafe"

	"github.com/viant/xunsafe"
)

// ReconcileData reconciles target with cached and predicted data
// target is either the pointer to the result or a pointer to a slice of results from
// the prediction server
func reconcileData(prefix string, target interface{}, cachable Cachable, cached []interface{}) error {
	targetType := reflect.TypeOf(target).Elem()
	targetPtr := xunsafe.AsPointer(target)

	if prefix != "" {
		log.Printf("%s reconciling: %T %+v", prefix, target, target)
	}

	switch targetType.Kind() {
	case reflect.Struct:
		if !cachable.CacheHit(0) {
			// the target memory already has actual value
			return nil
		}

		// directly replace the target memory with cached value
		*(*unsafe.Pointer)(targetPtr) = *(*unsafe.Pointer)(xunsafe.AsPointer(cached[0]))
		return nil
	case reflect.Slice:
		// noop
	default:
		return fmt.Errorf("unsupported target type expected *T or []*T, but had: %T", target)
	}

	// create a new slice since target slice needs to incorporate cache data
	xSlice := xunsafe.NewSlice(targetType.Elem())
	batchSize := cachable.BatchSize()

	// copy all cache values (including nils)
	newData := reflect.MakeSlice(xSlice.Type, batchSize, batchSize)
	newDataPtr := xunsafe.ValuePointer(&newData)
	hadDataOnlyInCache := len(cached) > 0
	for i, cacheEntry := range cached {
		if cacheEntry == nil {
			hadDataOnlyInCache = false
		}

		itemPtrAddr := xSlice.PointerAt(newDataPtr, uintptr(i))
		*(*unsafe.Pointer)(itemPtrAddr) = xunsafe.AsPointer(cached[i])

		if prefix != "" {
			log.Printf("%s cache->output[%v] %+v", prefix, i, cacheEntry)
		}
	}

	oldSlice := (*reflect.SliceHeader)(targetPtr)
	if !hadDataOnlyInCache {
		// copy all predicted values to nil spots from cache
		offsets := buildOffsets(batchSize, cachable)
		if prefix != "" {
			log.Printf("%s offsets map: %+v, oldSlice.Len:%d", prefix, offsets, oldSlice.Len)
		}

		for index := 0; index < oldSlice.Len; index++ {
			value := xSlice.ValuePointerAt(targetPtr, index)
			cacheableIndex := offsets[index]
			if cachable.CacheHit(cacheableIndex) {
				if prefix != "" {
					log.Printf("%s cache hit %d for index:%d", prefix, cacheableIndex, index)
				}

				continue
			}

			itemPtrAddr := xSlice.PointerAt(newDataPtr, uintptr(cacheableIndex))
			*(*unsafe.Pointer)(itemPtrAddr) = xunsafe.AsPointer(value)

			if prefix != "" {
				// means mly server response
				log.Printf("%s response[%v]->output[%v]: %+v", prefix, index, cacheableIndex, value)
			}
		}
	}

	nextSlice := *(*reflect.SliceHeader)(newDataPtr)
	oldSlice.Cap = nextSlice.Cap
	oldSlice.Len = nextSlice.Len
	oldSlice.Data = nextSlice.Data

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
