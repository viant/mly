package client

import (
	"github.com/jessevdk/go-flags"
	"log"
)

func Run(args []string) {
	options := &Options{}
	_, err := flags.ParseArgs(options, args)
	if err != nil {
		log.Fatal(err)
	}
	if err = RunWithOptions(options); err != nil {
		log.Fatal(err)
	}
}
