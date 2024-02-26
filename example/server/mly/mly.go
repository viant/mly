package main

import (
	"log"
	"os"

	"github.com/viant/mly/example/server"
)

var Version = "dev"

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	server.RunApp(Version, os.Args[1:])
}
