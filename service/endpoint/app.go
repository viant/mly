package endpoint

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jessevdk/go-flags"
)

type configProvider func(*Options) (*Config, error)

// Deprecated: use RunAppError
func RunApp(version string, args []string, wg *sync.WaitGroup) {
	err := RunAppError(version, args, wg)
	if err != nil {
		log.Fatal(err)
	}
}

func RunAppError(version string, args []string, wg *sync.WaitGroup) error {
	return RunAppWithConfigWaitError(version, args, func(o *Options) (*Config, error) {
		ctx := context.Background()
		return NewConfigFromURL(ctx, o.ConfigURL)
	}, wg)
}

// Deprecated: use RunAppErrorWithConfigError
func RunAppWithConfig(version string, args []string, cp configProvider) {
	err := RunAppWithConfigError(version, args, cp)
	if err != nil {
		log.Fatal(err)
	}
}

func RunAppWithConfigError(version string, args []string, cp configProvider) error {
	return RunAppWithConfigWaitError(version, args, cp, nil)
}

// RunAppWithConfigWaitError is the full options versions.
/// version is printed if provided in the options.
// TODO auto-determine version.
// cp is a function that can provide a configuration file.
// wg.Done() will be called once, when the server is finished booting up.
func RunAppWithConfigWaitError(version string, args []string, cp configProvider, wg *sync.WaitGroup) error {
	options := &Options{}
	_, err := flags.ParseArgs(options, args)
	if err != nil {
		return err
	}
	if IsHelpOption(args) {
		return nil
	}
	if options.Version {
		log.Printf("Mly: Version: %v\n", version)
		return nil
	}
	config, err := cp(options)
	if err != nil {
		return err
	}
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

	server := make(chan error)

	startServerWg := new(sync.WaitGroup)
	startServerWg.Add(1)

	var startErr error
	// run server in background
	go func() {
		l, err := srv.Listen()
		if err != nil {
			startErr = err
			startServerWg.Done()
			return
		}

		startServerWg.Done()
		server <- srv.Serve(l)
	}()

	startServerWg.Wait()

	if startErr != nil {
		return startErr
	}

	err = srv.SelfTest()
	if err != nil {
		defer srv.Shutdown(context.Background())
		return fmt.Errorf("self start test failure: %w", err)
	}

	log.Printf("self start test done, startup full time: %s", time.Now().Sub(start))

	if wg != nil {
		wg.Done()
	}

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
