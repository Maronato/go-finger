package main

import (
	"fmt"
	"os"

	"git.maronato.dev/maronato/finger/cmd"
)

// Version of the app.
var version = "dev"

func main() {
	// Run the server
	if err := cmd.Run(version); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
