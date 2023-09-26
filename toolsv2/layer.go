package main

import (
	"fmt"
	"io"
	"strings"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/mly/service/tfmodel"
)

func DiscoverLayers(model *tf.SavedModel, writer io.Writer) error {
	graph := model.Graph
	var exportables, consts []string
	mhtToLookup := make(map[string]string)
	for _, candidate := range graph.Operations() {
		name := candidate.Name()
		opType := candidate.Type()

		numOuts := candidate.NumOutputs()
		fmt.Printf("- %s type:%s numOutputs:%d\n", name, candidate.Type(), numOuts)
		for o := 0; o < numOuts; o++ {
			co := candidate.Output(o)
			fmt.Printf("  - %s.Output(%d) type:%v shape:%v\n", name, o, co.DataType(), co.Shape())
		}

		if strings.Contains(opType, "LookupTableExportV2") {
			exportables = append(exportables, name)
		}

		if opType == "Const" {
			consts = append(consts, name)
		}

		if strings.Contains(opType, "MutableHashTable") {
			mhtToLookup[name] = fmt.Sprintf("%s_lookup_table_export_values/LookupTableExportV2", name)
		}
	}

	fmt.Printf("--- %d LookupTableExportV2 ---\n", len(exportables))

	for _, exportable := range exportables {
		exported, err := tfmodel.GetAndRunExport(model.Session, graph, exportable)
		if err != nil {
			return err
		}

		var etl int
		switch et := exported.(type) {
		case []string:
			etl = len(et)
		default:
		}

		fmt.Printf("%s - %T - %d\n", exportable, exported, etl)
	}

	fmt.Printf("--- %d MutableHashTable ---\n", len(mhtToLookup))

	for mhtName, lookupName := range mhtToLookup {
		mhtOp := graph.Operation(mhtName)
		if mhtOp == nil {
			return fmt.Errorf("could not find operation %s", mhtName)
		}

		lookupOp := graph.Operation(lookupName)
		if lookupOp == nil {
			return fmt.Errorf("could not find lookup operation %s", lookupName)
		}

		feeds := make(map[tf.Output]*tf.Tensor)
		fetches := []tf.Output{mhtOp.Output(0)}
		targets := []*tf.Operation{mhtOp}
		output, err := model.Session.Run(feeds, fetches, targets)
		if err != nil {
			return fmt.Errorf("%s error:%w", mhtName, err)
		}

		fmt.Printf("MutableHashTable Run: %v\n", output[0])

		feeds = make(map[tf.Output]*tf.Tensor)
		feeds[mhtOp.Output(0)] = output[0]
		fetches = []tf.Output{lookupOp.Output(0)}
		targets = []*tf.Operation{lookupOp}
		output, err = model.Session.Run(feeds, fetches, targets)
		if err != nil {
			fmt.Println(fmt.Errorf("%s error:%w", lookupName, err))
			continue
		}

		fmt.Printf("MutableHashTable Lookup Run: %v\n", output[0].Value())
	}

	fmt.Printf("--- %d Const ---\n", len(consts))

	for _, constName := range consts {
		exported, err := tfmodel.GetAndRunExport(model.Session, graph, constName)
		if err != nil {
			return err
		}

		var etl int
		switch et := exported.(type) {
		case []string:
			etl = len(et)
		default:
		}

		fmt.Printf("%s - %T - %d\n", constName, exported, etl)
	}

	return nil
}
