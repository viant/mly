package log

import (
	"fmt"
	"os"
)

var debugEnabled = os.Getenv("DEBUG") == "true"

func Debug(temp string, args ...interface{}) {
	if ! debugEnabled {
		return
	}
	fmt.Printf(temp, args...)
}
