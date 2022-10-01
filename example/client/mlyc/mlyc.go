package main

import (
	"github.com/viant/mly/example/client"
	"os"
)

func main() {
	os.Args = []string{
		"", "-m=vec", "-t=a", "-t=b", "-s=d", "-s=d",
	}
	client.Run(os.Args[1:])
}
