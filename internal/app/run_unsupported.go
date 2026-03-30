//go:build !windows

package app

import (
	"fmt"
	"io"
	"runtime"
)

func run(_ Config, _ io.Writer, _ io.Writer) error {
	return fmt.Errorf("unsupported platform %s: virtcap currently only supports Windows", runtime.GOOS)
}
