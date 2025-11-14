package shape

import "fmt"

// DetermineBatchSize determines batch size from a service.Request.Feeds slice.
func DetermineBatchSize(inputs []interface{}) (int, error) {
	var batchSize int
	for _, iSlice := range inputs {
		switch typedSlice := iSlice.(type) {
		case [][]int32:
			batchSize = len(typedSlice)
		case [][]int64:
			batchSize = len(typedSlice)
		case [][]float32:
			batchSize = len(typedSlice)
		case [][]float64:
			batchSize = len(typedSlice)
		case [][]string:
			batchSize = len(typedSlice)
		default:
			continue
		}

		break
	}

	var err error
	if batchSize == 0 {
		err = fmt.Errorf("could not determine batch size")
	}

	return batchSize, err
}
