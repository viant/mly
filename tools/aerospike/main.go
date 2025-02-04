package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aerospike/aerospike-client-go"
	"github.com/jessevdk/go-flags"
	"github.com/viant/mly/example/req"
	"github.com/viant/mly/example/transformer/slf/model"
	"github.com/viant/mly/shared/client"
	"github.com/viant/mly/shared/config/datastore"
	dscli "github.com/viant/mly/shared/datastore/client"
)

var letters = "abcdefghijklmnopqrstuvwxyz"

type optst struct {
	ConnSharingEnabled bool `long:"connectionSharing"`
	NumClients         int  `long:"numClients"`
	Port               int  `long:"port"`

	ChannelSize int `long:"channelSize"`
	KeyRangeLen int `long:"keyRangeLen"`

	AerospikeMaxConnections int `long:"asMaxConns"`

	RunDurationSec int `long:"runDurationSec"`
	PostIdleSec    int `long:"postIdleSec"`
	PreIdleSec     int `long:"preIdleSec"`
}

func main() {
	opts := &optst{}
	_, err := flags.ParseArgs(opts, os.Args)
	if err != nil {
		panic(err)
	}

	numClients := opts.NumClients
	if numClients == 0 {
		numClients = 200
	}

	port := opts.Port
	if port == 0 {
		port = 8086
	}

	if opts.PreIdleSec == 0 {
		opts.PreIdleSec = 5
	}

	if opts.RunDurationSec == 0 {
		opts.RunDurationSec = 15
	}

	if opts.PostIdleSec == 0 {
		opts.PostIdleSec = 2
	}
	hosts := []*client.Host{client.NewHost("localhost", port)}

	ctx, cancel := context.WithCancel(context.Background())
	payloads := make(chan *req.SLF, opts.ChannelSize)

	options := []client.Option{
		client.WithDebug(true),
	}

	if opts.ConnSharingEnabled {
		conns := make(map[string]*dscli.Service)
		options = append(options, client.WithConnectionSharing(conns))
	}

	if opts.AerospikeMaxConnections > 0 {
		policy := aerospike.NewClientPolicy()
		policy.ConnectionQueueSize = opts.AerospikeMaxConnections
		dsOptPolicy := dscli.WithClientPolicy(policy)
		options = append(options, client.WithClientOptions(dsOptPolicy))
	}

	krl := opts.KeyRangeLen
	if krl == 0 {
		krl = 2
	}

	var runsDone, errorCount, errorConnResetByPeer uint64

	errorSlice := make([]error, 0)
	errorCollect := make(chan error, numClients)

	go func(es *[]error) {
		for {
			select {
			case err := <-errorCollect:
				*es = append(*es, err)
			case <-ctx.Done():
				return
			}
		}
	}(&errorSlice)

	var connInfo *datastore.Connection

	for i := 0; i < numClients; i++ {
		cli, err := client.New(
			"slf",
			hosts,
			options...,
		)

		if err != nil {
			panic(err)
		}

		if cli != nil && connInfo == nil {
			connInfo = cli.Config.Datastore.Connections[0]
		}

		go func() {
			for {
				select {
				case payload := <-payloads:
					msg := cli.NewMessage()
					payload.ToMessage(msg)
					response := &client.Response{
						Data: new(model.Segmented),
					}

					err = cli.Run(ctx, msg, response)
					if err != nil {
						fmt.Printf("error: %v\n", err)

						if errors.Is(err, context.Canceled) {
							return
						}

						if strings.Contains(err.Error(), "connection reset by peer") {
							atomic.AddUint64(&errorConnResetByPeer, 1)
							return
						}

						errorCollect <- err
						atomic.AddUint64(&errorCount, 1)
					}
					atomic.AddUint64(&runsDone, 1)

				case <-ctx.Done():
					return
				}
			}
		}()
	}

	fmt.Printf("idle %d seconds, %d clients\n", opts.PreIdleSec, numClients)

	getConnections := func() (int, error) {
		cPol := aerospike.NewClientPolicy()
		cPol.IdleTimeout = 5 * time.Second

		dsCli, err := dscli.NewWithOptions(connInfo, dscli.WithClientPolicy(cPol))

		if err != nil {
			return -1, err
		}

		asCluster := dsCli.Cluster()
		if asCluster == nil {
			fmt.Printf("asCluster is nil\n")
			return -1, nil
		}

		node, err := asCluster.GetRandomNode()
		if err != nil {
			return -1, err
		}

		if node == nil {
			fmt.Printf("node is nil\n")
			return -1, nil
		}

		ip := aerospike.NewInfoPolicy()
		ip.Timeout = 5 * time.Second
		info, err := node.RequestInfo(ip, "statistics")
		if err != nil {
			return -1, err
		}

		statsSSV := info["statistics"]
		for _, line := range strings.Split(statsSSV, ";") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 && parts[0] == "client_connections" {
				return strconv.Atoi(parts[1])
			}
		}

		return -1, nil
	}

	var idleConns, activeConns, postConns []int

	stopConnCounter := make(chan struct{})

	connCounterGo := func(collection *[]int) {
		for {
			idleConns, err := getConnections()
			if err != nil {
				fmt.Printf("error: %v\n", err)
			} else {
				*collection = append(*collection, idleConns)
			}

			select {
			case <-stopConnCounter:
				return
			default:
				time.Sleep(time.Second)
			}
		}
	}

	go connCounterGo(&idleConns)

	for i := 0; i < opts.PreIdleSec; i++ {
		time.Sleep(time.Second)
	}

	stopConnCounter <- struct{}{}

	fmt.Printf("run %d seconds, %d clients\n", opts.RunDurationSec, numClients)

	var payloadsQueued uint64

	go connCounterGo(&activeConns)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				sa := make([]byte, krl)
				sl := make([]byte, krl)
				for i := range sa {
					sa[i] = letters[rand.Intn(len(letters))]
					sl[i] = letters[rand.Intn(len(letters))]
				}

				payloads <- &req.SLF{SA: string(sa), SL: string(sl), Aux: "aux"}
				atomic.AddUint64(&payloadsQueued, 1)
			}
		}
	}()

	for i := 0; i < opts.RunDurationSec; i++ {
		time.Sleep(time.Second)
	}

	stopConnCounter <- struct{}{}

	cancel()

	go connCounterGo(&postConns)

	fmt.Printf("idle %d seconds, %d clients\n", opts.PostIdleSec, numClients)

	for i := 0; i < opts.PostIdleSec; i++ {
		time.Sleep(time.Second)
	}

	stopConnCounter <- struct{}{}

	fmt.Printf("payloads queued: %d, runs done: %d, errors: %d, errorConnResetByPeer: %d\n", payloadsQueued, runsDone, errorCount, errorConnResetByPeer)

	fmt.Printf("idleConns: %v\n", idleConns)
	fmt.Printf("activeConns: %v\n", activeConns)
	fmt.Printf("postConns: %v\n", postConns)

	fmt.Printf("errors:%v\n", errorSlice)
}
