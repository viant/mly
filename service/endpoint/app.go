package endpoint

import (
	"context"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/viant/mly/shared/pb"
	"google.golang.org/grpc"
	"log"
	"net"
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
	if err := config.Validate();err != nil {
		log.Fatal(err)
	}
	srv, err := New(config)
	if err != nil {
		log.Fatal(err)
	}
	if wg != nil {
		wg.Done()
	}
	if config.Endpoint.GRPCPort > 0 {
		go startGrpc(srv, config)
	}

	srv.ListenAndServe()
}

func startGrpc(srv *Service, config *Config) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Endpoint.GRPCPort))
	if err != nil {
		log.Fatalf("failed to GRPC listen: %v", err)
	}
	grpcServer := grpc.NewServer(grpc.MaxRecvMsgSize(64 *1024), grpc.MaxSendMsgSize(64 *1024))
	pb.RegisterEvaluatorServer(grpcServer, srv)
	fmt.Printf("starting mly GRCP endpoint: %v\n", config.Endpoint.GRPCPort)
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("failed to start GRPC: %v", err)
	}
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
