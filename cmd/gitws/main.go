package main

import (
	"fmt"
	"os"

	"github.com/gitworkspaces/gitws/internal/cli"
)

var version = "dev"

func main() {
	fmt.Fprintf(os.Stderr, "DEBUG: gitws starting, version: %s\n", version)
	fmt.Fprintf(os.Stderr, "DEBUG: args: %v\n", os.Args)

	if err := cli.Execute(version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "DEBUG: gitws completed successfully\n")
}
