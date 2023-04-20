package main

import (
	"context"
	"fmt"

	"github.com/viant/afs"
	"github.com/viant/mly/service/endpoint"
	"github.com/viant/mly/service/tfmodel"
	"github.com/viant/mly/tools"
)

// use this for to check is a command is being used
type Commandable struct {
	Active bool
}

type ConfigCommand struct {
	Commandable
	ID string `long:"id" description:"Model ID"`
}

// this will bind to the parser and mark the command as the active command, enabling "2-phase interpretation"
func (c *Commandable) Execute(args []string) error {
	c.Active = true
	return nil
}

type FlagSpec struct {
	Dict struct {
		JSON struct {
			URL string
		} `positional-args:"yes" required:"1" description:"Dictionary JSON file URL"`
	} `command:"dict" description:"Read a dictionary JSON file"`
	FromModel struct {
		Desc struct {
			Action string `short:"a" long:"action" required:"1" choice:"dict" choice:"signature" choice:"layers" description:"\n dict - Get Dictionary Hash\n signature - Show TFServe signatures\n layers - Show model layers"`
		} `command:"desc" description:"Print information about model"`

		Config struct {
			Commandable
			ID string `long:"id" description:"Model ID"`
		} `command:"config" description:"Generate a basic configuration file based on the model"`

		Table struct {
			Commandable
			Name   string `short:"t" long:"table" description:"GBQ full table name" default:"<TABLE IDENTIFIER>"`
			Single bool   `long:"single"`
		} `command:"table" description:"Generate columns for mly log"`

		ModelURL  string `short:"m" long:"model" required:"1" description:"Model URL"`
		OutputURL string `short:"o" long:"output" description:"Optional override on where to write output, defaults to STDOUT"`
	} `command:"discover" description:"Use model metadata ..."`
	Run struct {
		Config struct {
			URL string
		} `positional-args:"yes" required:"1" description:"Configuration file URL"`
	} `command:"run" description:"Run a server given a configuration file"`
}

func Operate(options *FlagSpec) error {
	if options.Run.Config.URL != "" {
		cfg, err := endpoint.NewConfigFromURL(context.Background(), options.Run.Config.URL)
		if err != nil {
			return err
		}

		srv, err := endpoint.New(cfg)
		if err != nil {
			return err
		}

		srv.ListenAndServe()
	} else if options.Dict.JSON.URL != "" {

	} else if options.FromModel.ModelURL != "" {
		discover := options.FromModel
		modelURL := discover.ModelURL
		model, err := tools.LoadModel(context.Background(), modelURL)
		if err != nil {
			return err
		}

		fs := afs.New()
		writer, err := tools.GetWriter(discover.OutputURL, fs)
		if err != nil {
			return err
		}

		signature, err := tfmodel.Signature(model)
		if err != nil {
			return err
		}

		if discover.Desc.Action != "" {
			// TODO these should all be commands
			switch discover.Desc.Action {
			case "dict":
				return tools.DiscoverDictHash(model, writer)
			case "signature":
				return tools.DiscoverSignature(writer, signature)
			case "layers":
				fmt.Println(model, writer)
			default:
				fmt.Printf("unknown action %s\n", discover.Desc.Action)
			}
		} else if discover.Config.Active {
			return tools.DiscoverConfig(modelURL, model, writer)
		} else if discover.Table.Active {
			return tools.GenerateTable(writer, discover.Table.Single, signature)
		} else {
			return fmt.Errorf("unrecognized command")
		}
	} else {
		return fmt.Errorf("unrecognized state")
	}

	return nil
}
