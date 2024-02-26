package signature

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/mly/service/domain"
)

// Signature searches the Tensorflow operation graph for inputs and outputs.
func Signature(model *tf.SavedModel) (*domain.Signature, error) {
	tfSignature, ok := model.Signatures[domain.DefaultSignatureKey]
	if !ok {
		return nil, fmt.Errorf("failed to lookup signature: %v", domain.DefaultSignatureKey)
	}

	sig := &domain.Signature{
		Method: tfSignature.MethodName,
	}

	for layerName, tfInfo := range tfSignature.Outputs {
		output := domain.Output{
			Name: layerName,
		}

		// tfInfo.Name is the Tensor name
		operationName := tfInfo.Name
		if index := strings.Index(operationName, ":"); index != -1 {
			indexValue := operationName[index+1:]
			operationName = operationName[:index]
			output.Index, _ = strconv.Atoi(indexValue)
		}

		if output.Operation = model.Graph.Operation(operationName); output.Operation == nil {
			return nil, fmt.Errorf("failed to lookup output operation '%v' for output: %v", operationName, layerName)
		}

		tryAssignDataType(tfInfo, &output)
		sig.Outputs = append(sig.Outputs, output)
	}

	sig.Output = sig.Outputs[0]

	sigInputs := tfSignature.Inputs
	var inputs = make([]string, 0, len(sigInputs))
	for k := range sigInputs {
		inputs = append(inputs, k)
	}

	sort.Strings(inputs)
	for _, k := range inputs {
		tfInfo := sigInputs[k]
		operationName := domain.DefaultSignatureKey + "_" + k
		operation := model.Graph.Operation(operationName)
		if operation == nil {
			return nil, fmt.Errorf("failed to lookup input operation: %v", operationName)
		}

		sig.Inputs = append(sig.Inputs, domain.Input{
			Name:        k,
			Index:       len(sig.Inputs),
			Type:        tf.TypeOf(tfInfo.DType, []int64{}),
			Placeholder: operation.Output(0),
		})
	}

	return sig, nil
}

func tryAssignDataType(v tf.TensorInfo, output *domain.Output) {
	defer func() {
		_ = recover()
	}()

	oType := tf.TypeOf(v.DType, []int64{})
	output.DataType = oType.Name()
	output.DataTypeKind = oType.Kind()
}
