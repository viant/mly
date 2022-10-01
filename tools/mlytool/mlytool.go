package main

import (
	"github.com/viant/mly/tools"
	"os"
)

func main() {
	tools.Run(os.Args[1:])
}
