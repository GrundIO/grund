package main

import (
	"os"

	"github.com/Saturn-Fintech/grund/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
