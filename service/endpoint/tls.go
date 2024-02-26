package endpoint

import (
	"fmt"
	"os"
)

type TLSConfig struct {
	CertFile string
	KeyFile  string
}

func (t *TLSConfig) Valid() error {
	if t.CertFile == "" {
		return fmt.Errorf("CertFile not set")
	}

	if t.KeyFile == "" {
		return fmt.Errorf("KeyFile not set")
	}

	fp, err := os.Open(t.CertFile)
	if err != nil {
		fp.Close()
	} else {
		return fmt.Errorf("could not open %s, %w", t.CertFile, err)
	}

	fp, err = os.Open(t.KeyFile)
	if err != nil {
		fp.Close()
	} else {
		return fmt.Errorf("could not open %s, %w", t.KeyFile, err)
	}

	return nil
}
