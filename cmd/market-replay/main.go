package main

import (
	"os"

	"sixletters/back-to-the-past/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
