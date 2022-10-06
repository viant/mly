package main

import (
	"github.com/viant/mly/tools"
	"os"
)

func main() {
	os.Args = []string{"", "-m=run", "-c=/Users/awitas/go/src/github.vianttech.com/adelphic/mly/kpi/vcr/config.yaml"}
	tools.Run(os.Args[1:])
}
