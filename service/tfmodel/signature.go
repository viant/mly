package tfmodel

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/mly/service/domain"
)

//Signature returns model signature or error
func Signature(model *tf.SavedModel) (*domain.Signature, error) {
	signature, ok := model.Signatures[domain.DefaultSignatureKey]
	if !ok {
		return nil, fmt.Errorf("failed to lookup signature: %v", domain.DefaultSignatureKey)
	}

	result := &domain.Signature{
		Method: signature.MethodName,
	}

	for k, v := range signature.Outputs {
		output := domain.Output{}
		output.Name = k
		operationName := v.Name
		if index := strings.Index(operationName, ":"); index != -1 {
			indexValue := operationName[index+1:]
			operationName = operationName[:index]
			output.Index, _ = strconv.Atoi(indexValue)
		}
		if output.Operation = model.Graph.Operation(operationName); output.Operation == nil {
			return nil, fmt.Errorf("failed to lookup operation '%v' for output: %v", operationName, k)
		}
		tryAssignDataType(v, &output)
		result.Outputs = append(result.Outputs, output)
	}

	result.Output = result.Outputs[0]
	var inputs = make([]string, 0, len(signature.Inputs))
	for k := range signature.Inputs {
		inputs = append(inputs, k)
	}
	sort.Strings(inputs)
	for _, k := range inputs {
		v := signature.Inputs[k]
		operationName := domain.DefaultSignatureKey + "_" + k
		operation := model.Graph.Operation(operationName)
		if operation == nil {
			return nil, fmt.Errorf("failed to lookup placeholder operation: %v", operationName)
		}

		result.Inputs = append(result.Inputs, domain.Input{
			Name:        k,
			Index:       len(result.Inputs),
			Type:        tf.TypeOf(v.DType, []int64{}),
			Placeholder: operation.Output(0),
		})
	}
	return result, nil
}

func tryAssignDataType(v tf.TensorInfo, output *domain.Output) {
	defer func() {
		_ = recover()
	}()

	oType := tf.TypeOf(v.DType, []int64{})
	output.DataType = oType.Name()
	output.DataTypeKind = oType.Kind()
}
