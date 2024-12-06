package tools

import (
	"context"
	"fmt"

	"github.com/jessevdk/go-flags"
	"github.com/viant/afs"
	tf "github.com/wamuir/graft/tensorflow"

	slog "log"
	"strings"

	"github.com/viant/mly/service/endpoint"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/service/tfmodel/signature"
	"github.com/viant/mly/shared/common"
	"github.com/viant/tapper/config"
	"github.com/viant/tapper/io"
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
		return FetchDictHash(writer, options.DestURL, fs)
	}

	model, err := LoadModel(context.Background(), options.SourceURL)
	if err != nil {
		return err
	}

	signature, err := signature.Signature(model)
	if err != nil {
		return err
	}

	switch options.Operation {
	case "signature":
		return DiscoverSignature(writer, signature)
	case "layers":
		return discoverLayers(options, model, int64(0), fs)
	case "config":
		return DiscoverConfig(options.SourceURL, model, writer)
	default:
		return fmt.Errorf("unsupported option: '%v'", options.Operation)
	}
}

// Deprecated: due to the changing of how hashing works, this is no longer supported or needed.
// TODO: make a different utility for checking vocabulary extraction.
func discoverLayers(options *Options, model *tf.SavedModel, fsh int64, fs afs.Service) error {
	if ok, _ := fs.Exists(context.Background(), options.DestURL); ok {
		_ = fs.Delete(context.Background(), options.DestURL)
	}

	var exportables []string
	for _, candidate := range model.Graph.Operations() {
		if strings.Contains(candidate.Name(), "LookupTableExportV2") {
			exportables = append(exportables, candidate.Name())
		}
	}

	dictionary, err := tfmodel.DiscoverDictionary(model.Session, model.Graph, exportables)
	if err != nil {
		return err
	}

	dictionary.UpdateHash(fsh)

	logger, err := log.New(&config.Stream{URL: options.DestURL}, "myID", afs.New())
	if err != nil {
		return err
	}

	provider := msg.NewProvider(100*1024*1024, 1, json.New)
	aMessage := provider.NewMessage()
	aMessage.PutInt("Hash", dictionary.Hash)
	aMessage.PutObjects("Layers", toEncoders(dictionary.Layers))
	logger.Log(aMessage)
	aMessage.Free()
	return logger.Close()
}

func toEncoders(l []common.Layer) []io.Encoder {
	var layers = make([]io.Encoder, len(l))
	for i := range l {
		layers[i] = &l[i]
	}
	return layers
}
