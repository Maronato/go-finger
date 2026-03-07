package main

import (
	"fmt"
	"os"

	"github.com/Maronato/go-finger/cmd"
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
