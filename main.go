package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/PeronGH/virtcap/internal/app"
)

func main() {
	if err := app.Main(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}

		fmt.Fprintf(os.Stderr, "virtcap: %v\n", err)
		os.Exit(1)
	}
}
