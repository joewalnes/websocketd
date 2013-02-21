package main

import (
	"flag"
	"fmt"
	"log"
)

type Config struct {
	Addr        string   // TCP address to listen on. e.g. ":1234", "1.2.3.4:1234"
	Verbose     bool     // Verbose logging.
	BasePath    string   // Base URL path. e.g. "/"
	CommandName string   // Command to execute.
	CommandArgs []string // Additional args to pass to command
}

func parseCommandLine() Config {
	var config Config

	portFlag := flag.Int("port", 80, "HTTP port to listen on (required)")
	addressFlag := flag.String("address", "0.0.0.0", "Interface to bind to (e.g. 127.0.0.1)")
	basePathFlag := flag.String("basepath", "/", "Base URL path (e.g /)")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")

	flag.Parse()

	config.Addr = fmt.Sprintf("%s:%d", *addressFlag, *portFlag)
	config.Verbose = *verboseFlag
	config.BasePath = *basePathFlag

	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("No executable command specified")
	}
	config.CommandName = args[0]
	config.CommandArgs = flag.Args()[1:]

	return config
}
