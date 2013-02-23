// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
)

type Config struct {
	Addr          string   // TCP address to listen on. e.g. ":1234", "1.2.3.4:1234"
	Verbose       bool     // Verbose logging.
	BasePath      string   // Base URL path. e.g. "/"
	CommandName   string   // Command to execute.
	CommandArgs   []string // Additional args to pass to command.
	ReverseLookup bool     // Perform reverse DNS lookups on hostnames (useful, but slower).
}

func parseCommandLine() Config {
	var config Config

	portFlag := flag.Int("port", 80, "HTTP port to listen on (required)")
	addressFlag := flag.String("address", "0.0.0.0", "Interface to bind to (e.g. 127.0.0.1)")
	basePathFlag := flag.String("basepath", "/", "Base URL path (e.g /)")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")
	reverseLookupFlag := flag.Bool("reverselookup", true, "Perform reverse DNS lookups on remote clients")

	flag.Parse()

	config.Addr = fmt.Sprintf("%s:%d", *addressFlag, *portFlag)
	config.Verbose = *verboseFlag
	config.BasePath = *basePathFlag
	config.ReverseLookup = *reverseLookupFlag

	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("No executable command specified")
	}
	config.CommandName = args[0]
	config.CommandArgs = flag.Args()[1:]

	return config
}
