package main

import (
	"log"
	"os"

	"github.com/viant/mly/example/server"
)

var Version = "dev"

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	err := server.RunApp(Version, os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}
