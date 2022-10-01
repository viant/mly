package main

import (
	"github.com/viant/mly/example/server"
	"os"
)

var Version = "dev"

func main() {
	server.RunApp(Version, os.Args[1:])
}
