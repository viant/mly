package tools

import (
	"context"
	"fmt"

	"github.com/jessevdk/go-flags"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/afs"

	slog "log"
	"strings"

	"github.com/viant/mly/service/endpoint"
	"github.com/viant/mly/service/layers"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/shared/common"
	"github.com/viant/tapper/config"
	"github.com/viant/tapper/log"
	"github.com/viant/tapper/msg"
	"github.com/viant/tapper/msg/json"
)

func Run(args []string) {
	options := &Options{}

	_, err := flags.ParseArgs(options, args)
	if err != nil {
		slog.Fatal(err)
	}
	if err := options.Validate(); err != nil {
		slog.Fatal(err)
	}

	switch options.Mode {
	case "discover":
		err = Discover(options)
		if err != nil {
			slog.Fatal(err)
		}
	case "run":
		cfg, err := endpoint.NewConfigFromURL(context.Background(), options.ConfigURL)
		if err != nil {
			slog.Fatal(err)
			return
		}
		srv, err := endpoint.New(cfg)
		if err != nil {
			slog.Fatal(err)
			return
		}

		srv.ListenAndServe()
	}
}

func Discover(options *Options) error {
	fs := afs.New()
	writer, err := GetWriter(options.DestURL, fs)
	if err != nil {
		return err
	}

	defer writer.Close()
	if options.Operation == "dictHash" {
		return FetchDictHash(options, fs, writer)
	}

	model, err := LoadModel(context.Background(), options.SourceURL)
	if err != nil {
		return err
	}

	signature, err := tfmodel.Signature(model)
	if err != nil {
		return err
	}

	switch options.Operation {
	case "signature":
		return DiscoverSignature(writer, signature)
	case "layers":
		return discoverLayers(options, model, fs)
	case "config":
		return DiscoverConfig(options.SourceURL, model, writer)
	default:
		return fmt.Errorf("unsupported option: '%v'", options.Operation)
	}
}

func discoverLayers(options *Options, model *tf.SavedModel, fs afs.Service) error {
	if ok, _ := fs.Exists(context.Background(), options.DestURL); ok {
		_ = fs.Delete(context.Background(), options.DestURL)
	}
	var exportables []string
	for _, candidate := range model.Graph.Operations() {
		if strings.Contains(candidate.Name(), "LookupTableExportV2") {
			exportables = append(exportables, candidate.Name())
		}
	}
	dictionary, err := layers.DiscoverDictionary(model.Session, model.Graph, exportables)
	if err != nil {
		return err
	}

	logger, err := log.New(&config.Stream{URL: options.DestURL}, "myID", afs.New())
	if err != nil {
		return err
	}

	provider := msg.NewProvider(100*1024*1024, 1, json.New)
	aMessage := provider.NewMessage()
	dictLayers := common.Layers(dictionary.Layers)

	aMessage.PutInt("Hash", dictionary.Hash)
	aMessage.PutObjects("Layers", dictLayers.Encoders())
	logger.Log(aMessage)
	aMessage.Free()
	return logger.Close()
}
