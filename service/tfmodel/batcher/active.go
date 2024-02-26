package batcher

import "fmt"

func modifyInterfaceSlice(active []interface{}, other []interface{}) (err error) {
	for o, inSlice := range other {
		if active[o] == nil {
			active[o] = inSlice
			continue
		}

		// TODO maybe use xSlice
		switch slicedActive := active[o].(type) {
		case [][]int32:
			switch aat := inSlice.(type) {
			case [][]int32:
				for _, untyped := range aat {
					slicedActive = append(slicedActive, untyped)
				}
			default:
				err = fmt.Errorf("inner expected int32 got %v", aat)
				break
			}
			active[o] = slicedActive
		case [][]int64:
			switch aat := inSlice.(type) {
			case [][]int64:
				for _, untyped := range aat {
					slicedActive = append(slicedActive, untyped)
				}
			default:
				err = fmt.Errorf("inner expected int64 got %v", aat)
				break
			}
			active[o] = slicedActive
		case [][]float32:
			switch aat := inSlice.(type) {
			case [][]float32:
				for _, untyped := range aat {
					slicedActive = append(slicedActive, untyped)
				}
			default:
				err = fmt.Errorf("inner expected float32 got %v", aat)
				break
			}
			active[o] = slicedActive
		case [][]float64:
			switch aat := inSlice.(type) {
			case [][]float64:
				for _, untyped := range aat {
					slicedActive = append(slicedActive, untyped)
				}
			default:
				err = fmt.Errorf("inner expected float64 got %v", aat)
				break
			}
			active[o] = slicedActive
		case [][]string:
			switch aat := inSlice.(type) {
			case [][]string:
				for _, untyped := range aat {
					slicedActive = append(slicedActive, untyped)
				}
			default:
				err = fmt.Errorf("inner expected string got %v", aat)
				break
			}
			active[o] = slicedActive
		default:
			err = fmt.Errorf("unexpected initial value: %v", slicedActive)
			break
		}
	}

	return err
}
