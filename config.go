package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
)

type Config struct {
	Addr           string   // TCP address to listen on. e.g. ":1234", "1.2.3.4:1234"
	Verbose        bool     // Verbose logging.
	BasePath       string   // Base URL path. e.g. "/"
	CommandName    string   // Command to execute.
	CommandArgs    []string // Additional args to pass to command.
	ReverseLookup  bool     // Perform reverse DNS lookups on hostnames (useful, but slower).
	ScriptDir      string   // Base directory for websocket scripts
	UsingScriptDir bool     // Are we running with a script dir
}

func parseCommandLine() Config {
	var config Config

	portFlag := flag.Int("port", 80, "HTTP port to listen on (required)")
	addressFlag := flag.String("address", "0.0.0.0", "Interface to bind to (e.g. 127.0.0.1)")
	basePathFlag := flag.String("basepath", "/", "Base URL path (e.g /)")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose logging")
	reverseLookupFlag := flag.Bool("reverselookup", true, "Perform reverse DNS lookups on remote clients")
	scriptDirFlag := flag.String("scriptdir", "", "Base directory for websocket scripts")

	flag.Parse()

	config.Addr = fmt.Sprintf("%s:%d", *addressFlag, *portFlag)
	config.Verbose = *verboseFlag
	config.BasePath = *basePathFlag
	config.ReverseLookup = *reverseLookupFlag
	config.ScriptDir = *scriptDirFlag

	args := flag.Args()
	if len(args) < 1 && config.ScriptDir == "" {
		log.Fatal("Please specify a command OR a script dir")
	}

	if len(args) > 0 {
		if config.ScriptDir != "" {
			log.Fatal("Please specify a command OR a script dir")
		}
		config.CommandName = args[0]
		config.CommandArgs = flag.Args()[1:]
		config.UsingScriptDir = false
	}

	if len(config.ScriptDir) > 0 {
		scriptDir, err := filepath.Abs(config.ScriptDir)
		if err != nil {
			log.Fatal("Could not find", config.ScriptDir)
		}
		config.ScriptDir = scriptDir
		config.UsingScriptDir = true
	}

	return config
}
