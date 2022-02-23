package endpoint

import (
	"context"
	"github.com/jessevdk/go-flags"
	"log"
	"sync"
)

//RunAppWithConfig run application
func RunAppWithConfig(Version string, args []string, configProvider func(options *Options) (*Config, error)) {
	options := &Options{}
	_, err := flags.ParseArgs(options, args)
	if err != nil {
		log.Fatal(err)
	}
	if IsHelpOption(args) {
		return
	}
	if options.Version {
		log.Printf("Mly: Version: %v\n", Version)
		return
	}
	config, err := configProvider(options)
	if err != nil {
		log.Fatal(err)
	}
	runApp(config, nil)
}

//RunApp run application
func RunApp(Version string, args []string, wg *sync.WaitGroup) {
	options := &Options{}
	_, err := flags.ParseArgs(options, args)
	if err != nil {
		log.Fatal(err)
	}
	if IsHelpOption(args) {
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
	config.Init()
	runApp(config, wg)
}

func runApp(config *Config, wg *sync.WaitGroup) {
	if err := config.Validate(); err != nil {
		log.Fatal(err)
	}
	srv, err := New(config)
	if err != nil {
		log.Fatal(err)
	}
	if wg != nil {
		wg.Done()
	}
	srv.ListenAndServe()
}

//IsHelpOption returns true if helper
func IsHelpOption(args []string) bool {
	for _, arg := range args {
		if arg == "-h" {
			return true
		}
	}
	return false
}
