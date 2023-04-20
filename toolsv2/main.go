package main

import (
	"log"

	"github.com/jessevdk/go-flags"
)

func main() {
	options := new(FlagSpec)
	_, err := flags.Parse(options)
	if flags.WroteHelp(err) {
		return
	}

	if err != nil {
		log.Fatal(err)
	}

	err = Operate(options)
	if err != nil {
		log.Fatal(err)
	}
}
