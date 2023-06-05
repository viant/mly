package endpoint

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

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

	start := time.Now()

	srv, err := New(config)
	if err != nil {
		return err
	}
	if wg != nil {
		wg.Done()
	}

	server := make(chan error)

	m := sync.Mutex{}
	m.Lock()

	// run server in background
	go func() {
		l, err := srv.Listen()
		if err != nil {
			server <- err
		}

		m.Unlock()
		server <- srv.Serve(l)
	}()

	m.Lock()
	defer m.Unlock()

	err = srv.SelfTest()
	if err != nil {
		defer srv.Shutdown(context.Background())
		return fmt.Errorf("self start test failure: %w", err)
	}

	log.Printf("self start test done, startup full time: %s", time.Now().Sub(start))

	return <-server
}

// IsHelpOption returns true if helper
func IsHelpOption(args []string) bool {
	for _, arg := range args {
		if arg == "-h" {
			return true
		}
	}

	return false
}
