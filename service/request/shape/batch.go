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

func SqueezeBatch(untypedBatch interface{}) (interface{}, error) {
	switch typedBatch := untypedBatch.(type) {
	case [][]int32:
		return typedBatch[0][0], nil
	case [][]int64:
		return typedBatch[0][0], nil
	case [][]float32:
		return typedBatch[0][0], nil
	case [][]float64:
		return typedBatch[0][0], nil
	case [][]string:
		return typedBatch[0][0], nil
	}

	return nil, fmt.Errorf("unexpected batch type: %T", untypedBatch)
}

func Debatch(untypedBatch interface{}, i int) (interface{}, error) {
	switch typedBatch := untypedBatch.(type) {
	case [][]int32:
		return [][]int32{{typedBatch[i][0]}}, nil
	case [][]int64:
		return [][]int64{{typedBatch[i][0]}}, nil
	case [][]float32:
		return [][]float32{{typedBatch[i][0]}}, nil
	case [][]float64:
		return [][]float64{{typedBatch[i][0]}}, nil
	case [][]string:
		return [][]string{{typedBatch[i][0]}}, nil
	}

	return nil, fmt.Errorf("unexpected batch type: %T", untypedBatch)
}

// concatAxis0 concatenates two tensors along axis 0 (batch dimension).
func ConcatAxis0(x []interface{}, y []interface{}) ([]interface{}, error) {
	if len(x) != len(y) {
		return nil, fmt.Errorf("x and y must have the same length: %d vs %d", len(x), len(y))
	}

	result := make([]interface{}, len(x))
	for i := range x {
		xt := x[i]
		yt := y[i]

		if xt == nil {
			result[i] = yt
			continue
		}

		switch xv := xt.(type) {
		case [][]int32:
			yv, ok := yt.([][]int32)
			if !ok {
				return nil, fmt.Errorf("type mismatch at index %d: %T vs %T", i, xt, yt)
			}

			result[i] = append(xv, yv...)
		case [][]int64:
			yv, ok := yt.([][]int64)
			if !ok {
				return nil, fmt.Errorf("type mismatch at index %d: %T vs %T", i, xt, yt)
			}
			result[i] = append(xv, yv...)
		case [][]float32:
			yv, ok := yt.([][]float32)
			if !ok {
				return nil, fmt.Errorf("type mismatch at index %d: %T vs %T", i, xt, yt)
			}
			result[i] = append(xv, yv...)
		case [][]float64:
			yv, ok := yt.([][]float64)
			if !ok {
				return nil, fmt.Errorf("type mismatch at index %d: %T vs %T", i, xt, yt)
			}
			result[i] = append(xv, yv...)
		case [][]string:
			yv, ok := yt.([][]string)
			if !ok {
				return nil, fmt.Errorf("type mismatch at index %d: %T vs %T", i, xt, yt)
			}
			result[i] = append(xv, yv...)
		default:
			return nil, fmt.Errorf("unexpected output tensor type at index %d: %T", i, xt)
		}
	}

	return result, nil
}
