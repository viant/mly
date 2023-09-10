package tfmodel

import (
	"log"
	"os"
	"strconv"
	"time"
)

var TFSessionPanicDuration time.Duration = 1 * time.Minute

func init() {
	pd := os.Getenv("MLY_TF_SESSION_PANIC_DURATION_SECONDS")
	if pd != "" {
		k, err := strconv.Atoi(pd)
		if err != nil {
			log.Print(err)
			return
		}

		TFSessionPanicDuration = time.Duration(k) * time.Second
	}
}
