package endpoint

import (
	"context"
	"log"
	"sync"

	"github.com/jessevdk/go-flags"
)

//RunAppWithConfig run application
func RunAppWithConfig(Version string, args []string, configProvider func(options *Options) (*Config, error)) {
	err := RunAppWithConfigError(Version, args, configProvider)
	if err != nil {
		log.Fatal(err)
	}
}

func RunAppWithConfigError(Version string, args []string, configProvider func(options *Options) (*Config, error)) error {
	options := &Options{}
	_, err := flags.ParseArgs(options, args)
	if err != nil {
		return err
	}
	if IsHelpOption(args) {
		return nil
	}
	if options.Version {
		log.Printf("Mly: Version: %v\n", Version)
		return nil
	}
	config, err := configProvider(options)
	if err != nil {
		return err
	}
	return runApp(config, nil)
}

//RunApp run application
func RunApp(Version string, args []string, wg *sync.WaitGroup) {
	err := RunAppError(Version, args, wg)
	if err != nil {
		log.Fatal(err)
	}
}

func RunAppError(Version string, args []string, wg *sync.WaitGroup) error {
	options := &Options{}
	_, err := flags.ParseArgs(options, args)
	if err != nil {
		return nil
	}

	if IsHelpOption(args) {
		return nil
	}

	if options.Version {
		log.Printf("Mly: Version: %v\n", Version)
		return nil
	}

	ctx := context.Background()
	config, err := NewConfigFromURL(ctx, options.ConfigURL)
	if err != nil {
		return nil
	}
	config.Init()

	return runApp(config, wg)
}

func runApp(config *Config, wg *sync.WaitGroup) error {
	if err := config.Validate(); err != nil {
		return err
	}

	srv, err := New(config)
	if err != nil {
		return err
	}
	if wg != nil {
		wg.Done()
	}
	return srv.ListenAndServe()
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
