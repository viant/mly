package tfmodel

import (
	"fmt"
	"strings"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

const (
	operationExportTemplate = "%s_lookup_index_table_lookup_table_export_values/LookupTableExportV2"
)

// Export attempts to pull the embedded lookup table from the Tensorflow graph.
func Export(session *tf.Session, graph *tf.Graph, name string) (interface{}, error) {
	operationName := name
	if !strings.HasSuffix(name, "/LookupTableExportV2") {
		operationName = MatchOperation(graph, name)
		if operationName == "" {
			return nil, fmt.Errorf("failed to match operation for %v", name)
		}
	}

	return GetAndRunExport(session, graph, operationName)
}

func GetAndRunExport(session *tf.Session, graph *tf.Graph, operationName string) (interface{}, error) {
	expOperation := graph.Operation(operationName)
	if expOperation == nil {
		return nil, fmt.Errorf("RunExport: failed to find operation:%v", operationName)
	}

	return RunExport(session, expOperation)
}

// MatchOperation will attempt to locate an LookupTableExportV2 operation that is associated to the provided name.
func MatchOperation(graph *tf.Graph, name string) string {
	for _, candidate := range graph.Operations() {
		if strings.HasPrefix(candidate.Name(), name+"_") && strings.HasSuffix(candidate.Name(), "LookupTableExportV2") {
			return candidate.Name()
		}
	}

	// dump all operations if no match
	for _, candidate := range graph.Operations() {
		fmt.Printf(" - %s\n", candidate.Name())
	}

	return ""
}

//RunExport runs export
func RunExport(session *tf.Session, exportOp *tf.Operation) (interface{}, error) {
	// for LookupTableExportV2, output offset 1 is the encoded value
	ipOutput := exportOp.Output(0)
	fetches := []tf.Output{ipOutput}

	targets := []*tf.Operation{exportOp}

	feeds := map[tf.Output]*tf.Tensor{}
	output, err := session.Run(feeds, fetches, targets)
	if err != nil {
		return nil, err
	}

	value := output[0].Value()

	return value, nil
}
