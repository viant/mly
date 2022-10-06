package main

import (
	"github.com/viant/mly/example/client"
	"os"
)

func main() {
	client.Run(os.Args[1:])
}
