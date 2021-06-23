package tfmodel

import (
	"fmt"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"strings"
)

const (
	operationExportTemplate = "%s_lookup_index_table_lookup_table_export_values/LookupTableExportV2"
)

//Export RunExport model layer, or error
func Export(graph *tf.Graph, name string) (interface{}, error) {
	operationName := name
	if !strings.HasSuffix(name, "/LookupTableExportV2") {
		operationName = fmt.Sprintf(operationExportTemplate, name)
	}

	expOperation := graph.Operation(operationName)
	if expOperation == nil {
		return nil, fmt.Errorf("failed to lookup RunExport operation: %v", operationName)
	}
	return RunExport(graph, expOperation)
}


//RunExport runs export
func RunExport(graph *tf.Graph, exportOp *tf.Operation) (interface{}, error) {
	session, err := tf.NewSession(graph, &tf.SessionOptions{})
	if err != nil {
		return nil, err
	}
	defer session.Close()
	ipOutput := exportOp.Output(0)
	feeds := map[tf.Output]*tf.Tensor{}
	fetches := []tf.Output{ipOutput}
	targets := []*tf.Operation{exportOp}
	output, err := session.Run(feeds, fetches, targets)
	if err != nil {
		return nil, err
	}
	value := output[0].Value()
	return value, nil
}
