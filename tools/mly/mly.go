package main

import (
	_ "github.com/viant/afsc/gs"
	_ "github.com/viant/afsc/s3"
	"github.com/viant/mly/tools"
	"os"
)

func main() {
	tools.Run(os.Args[1:])
}
