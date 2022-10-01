package tfmodel

import (
	"fmt"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"strings"
)

const (
	operationExportTemplate = "%s_lookup_index_table_lookup_table_export_values/LookupTableExportV2"
)

//Export run model export, or error
func Export(session *tf.Session, graph *tf.Graph, name string) (interface{}, error) {
	operationName := name
	if !strings.HasSuffix(name, "/LookupTableExportV2") {
		operationName = MatchOperation(graph, name)
		if operationName == "" {
			return nil, fmt.Errorf("failed to match operation for %v", name)
		}
	}
	expOperation := graph.Operation(operationName)
	if expOperation == nil {
		if expOperation == nil {
			return nil, fmt.Errorf("failed to lookup RunExport operation: %v", operationName)
		}
	}
	return RunExport(session, expOperation)
}

func MatchOperation(graph *tf.Graph, name string) string {
	for _, candidate := range graph.Operations() {
		if strings.HasPrefix(candidate.Name(), name+"_") && strings.HasSuffix(candidate.Name(), "LookupTableExportV2") {
			return candidate.Name()
		}
	}
	return ""
}

//RunExport runs export
func RunExport(session *tf.Session, exportOp *tf.Operation) (interface{}, error) {
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
