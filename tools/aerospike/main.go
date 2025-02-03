package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/viant/mly/example/req"
	"github.com/viant/mly/example/transformer/slf/model"
	"github.com/viant/mly/shared/client"
	dscli "github.com/viant/mly/shared/datastore/client"
)

var letters = "abcdefghijklmnopqrstuvwxyz"

type optst struct {
	ConnSharingEnabled bool `long:"connectionSharing"`
	NumClients         int  `long:"numClients"`
	Port               int  `long:"port"`

	ChannelSize int `long:"channelSize"`
	KeyRangeLen int `long:"keyRangeLen"`

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
		opts.PostIdleSec = 5
	}
	hosts := []*client.Host{client.NewHost("localhost", port)}

	clients := make([]*client.Service, 0)
	ctx, cancel := context.WithCancel(context.Background())
	payloads := make(chan *req.SLF, opts.ChannelSize)

	options := []client.Option{
		client.WithDebug(true),
	}

	if opts.ConnSharingEnabled {
		conns := make(map[string]*dscli.Service)
		options = append(options, client.WithConnectionSharing(conns))
	}

	krl := opts.KeyRangeLen
	if krl == 0 {
		krl = 2
	}

	for i := 0; i < numClients; i++ {
		cli, err := client.New(
			"slf",
			hosts,
			options...,
		)

		if err != nil {
			panic(err)
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
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	fmt.Printf("idle %d seconds, %d clients\n", opts.PreIdleSec, len(clients))

	for i := 0; i < opts.PreIdleSec; i++ {
		time.Sleep(time.Second)
	}

	fmt.Printf("run %d seconds, %d clients\n", opts.RunDurationSec, len(clients))

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
			}
		}
	}()

	for i := 0; i < opts.RunDurationSec; i++ {
		time.Sleep(time.Second)
	}

	cancel()

	fmt.Printf("idle %d seconds, %d clients\n", opts.PostIdleSec, len(clients))

	for i := 0; i < opts.PostIdleSec; i++ {
		time.Sleep(time.Second)
	}

}
