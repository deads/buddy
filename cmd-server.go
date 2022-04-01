/// File: cmd-server.go
/// Purpose: A command-line interface (CLI) to the FIFO pipe-based
/// command server. It is barebones and intended to be one
/// of the few quickbuddy commands allowed on a container.

/// Author: Damian Eads
package main

import (
	"fmt"
	. "quickbuddy"
	"os"
)

/// The main entry point into the cmd-server program.
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr,
			`usage: cmd-server COMMAND-FILENAME
			Runs a command server using the named FIFO pipe COMMAND-FILENAME.`)
		os.Exit(1)
	}
	command_filename := os.Args[1]
	// Create a new FIFO object that will run our server. No resources
	// or files have been allocated yet.
	cmd := NewFIFOCommand(command_filename, "")
	// Run the server.
	err := cmd.RunServer()
	// Report any errors.
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmd-server error: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
