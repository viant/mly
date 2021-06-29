package endpoint

import (
	"context"
	"github.com/jessevdk/go-flags"
	"github.vianttech.com/adelphic/mediator/common"
	"log"
	"sync"
)

var Started sync.WaitGroup

//RunApp run application
func RunApp(Version string, args []string) {
	common.Version = Version
	options := &Options{}
	_, err := flags.ParseArgs(options, args)
	if err != nil {
		log.Fatal(err)
	}
	if isHelpOption(args) {
		return
	}
	if options.Version {
		log.Printf("Mly: Version: %v\n", Version)
		return
	}

	ctx := context.Background()
	config, err := NewConfigFromURL(ctx, options.ConfigURL)
	if err != nil {
		log.Fatal(err)
	}
	app, err := New(config)
	if err != nil {
		log.Fatal(err)
	}
	Started.Done()
	app.ListenAndServe()
}

func isHelpOption(args []string) bool {
	for _, arg := range args {
		if arg == "-h" {
			return true
		}
	}
	return false
}
