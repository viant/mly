package tools

import (
	"context"
	sjson "encoding/json"
	"fmt"
	"github.com/jessevdk/go-flags"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/viant/afs"
	"github.com/viant/afs/file"
	"github.com/viant/afs/url"
	sconfig "github.com/viant/mly/service/config"
	"github.com/viant/mly/service/domain"
	dconfig "github.com/viant/mly/shared/config"
	"github.com/viant/scache"

	"github.com/viant/mly/shared"
	"github.com/viant/mly/shared/config/datastore"

	"github.com/viant/mly/service/endpoint"
	"github.com/viant/mly/service/layers"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/shared/common"
	"github.com/viant/tapper/config"
	"github.com/viant/tapper/log"
	"github.com/viant/tapper/msg"
	"github.com/viant/tapper/msg/json"
	"gopkg.in/yaml.v3"
	"io"
	slog "log"
	"os"
	"path"
	"strings"
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
	err = Discover(options)
	if err != nil {
		slog.Fatal(err)
	}
}

func Discover(options *Options) error {
	model, err := loadModel(context.Background(), options.SourceURL)
	if err != nil {
		return err
	}
	fs := afs.New()
	switch options.Operation {
	case "signature":
		return discoverSignature(options, model, fs)
	case "layers":
		return discoverLayers(options, model, fs)
	case "config":
		return discoverConfig(options, model, fs)

	default:
		return fmt.Errorf("unsupported option: '%v'", options.Operation)
	}

}

const exportSuffix = "_lookup_index_table_lookup_table_export_values/LookupTableExportV2"

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
	aMessage.PutInt("Hash", 0)
	aMessage.PutObjects("Layers", dictLayers.Encoders())
	logger.Log(aMessage)
	aMessage.Free()
	return logger.Close()
}

func discoverConfig(options *Options, model *tf.SavedModel, fs afs.Service) error {
	signature, err := tfmodel.Signature(model)
	if err != nil {
		return err
	}
	_, ID := path.Split(options.SourceURL)

	cfg := buildDefaultConfig(options, model, ID, signature)

	writer, err := getWriter(options, fs)
	if err != nil {
		return err
	}
	defer writer.Close()
	encoder := yaml.NewEncoder(writer)
	return encoder.Encode(cfg)
}

func buildDefaultConfig(options *Options, model *tf.SavedModel, ID string, signature *domain.Signature) endpoint.Config {
	cfg := endpoint.Config{}
	cfg.Connections = append(cfg.Connections, &datastore.Connection{
		ID:        "l1",
		Hostnames: "127.0.0.1",
		Port:      3000,
	})
	cfg.Datastores = append(cfg.Datastores, &dconfig.Datastore{
		ID:    "mly_l1",
		Cache: &scache.Config{SizeMb: 128},
		Reference: &datastore.Reference{
			Connection:   "l1",
			Namespace:    "test",
			Dataset:      ID,
			TimeToLiveMs: 0,
			RetryTimeMs:  0,
			ReadOnly:     false,
		},
		L2:       nil,
		Storable: "",
		Fields:   nil,
		Disabled: false,
	})

	cfg.Endpoint.Port = 8087
	configModel := &sconfig.Model{}
	configModel.ID = ID
	useDict := true
	configModel.UseDict = &useDict
	configModel.URL = options.SourceURL
	configModel.DataStore = ID
	var fields []*shared.Field
	for _, input := range signature.Inputs {
		hasDictionary := tfmodel.MatchOperation(model.Graph, input.Name) != ""
		fields = append(fields, &shared.Field{
			Name:     input.Name,
			Index:    input.Index,
			Wildcard: !hasDictionary,
		})
	}
	configModel.Inputs = fields
	cfg.Models = []*sconfig.Model{configModel}
	return cfg
}

func discoverSignature(options *Options, model *tf.SavedModel, fs afs.Service) error {
	signature, err := tfmodel.Signature(model)
	if err != nil {
		return err
	}
	writer, err := getWriter(options, fs)
	if err != nil {
		return err
	}
	defer writer.Close()
	encoder := sjson.NewEncoder(writer)
	return encoder.Encode(signature)
}

func getWriter(options *Options, fs afs.Service) (io.WriteCloser, error) {
	var writer io.WriteCloser = os.Stdout
	var err error
	if options.DestURL != "" {
		if writer, err = fs.NewWriter(context.Background(), options.DestURL, file.DefaultFileOsMode); err != nil {
			return nil, err
		}
	}
	return writer, nil
}

func loadModel(ctx context.Context, URL string) (*tf.SavedModel, error) {
	fs := afs.New()
	location := url.Path(URL)
	if url.Scheme(URL, file.Scheme) != file.Scheme {
		_, name := path.Split(URL)
		location = path.Join(os.TempDir(), name)
		if err := fs.Copy(ctx, URL, location); err != nil {
			return nil, err
		}
	}
	model, err := tf.LoadSavedModel(location, []string{"serve"}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load model %v, due to %w", location, err)
	}
	return model, nil
}
