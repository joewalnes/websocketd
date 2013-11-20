// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joewalnes/websocketd/libwebsocketd"
)

type Config struct {
	BasePath string // Base URL path. e.g. "/"
	Addr     string // TCP address to listen on. e.g. ":1234", "1.2.3.4:1234"
	LogLevel libwebsocketd.LogLevel
	*libwebsocketd.Config
}

func parseCommandLine() Config {
	var mainConfig Config
	var config libwebsocketd.Config

	// If adding new command line options, also update the help text in help.go.
	// The flag library's auto-generate help message isn't pretty enough.

	// server config options
	portFlag := flag.Int("port", 80, "HTTP port to listen on")
	addressFlag := flag.String("address", "0.0.0.0", "Interface to bind to (e.g. 127.0.0.1)")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	licenseFlag := flag.Bool("license", false, "Print license and exit")
	logLevelFlag := flag.String("loglevel", "access", "Log level, one of: debug, trace, access, info, error, fatal")

	// lib config options
	basePathFlag := flag.String("basepath", "/", "Base URL path (e.g /)")
	reverseLookupFlag := flag.Bool("reverselookup", true, "Perform reverse DNS lookups on remote clients")
	scriptDirFlag := flag.String("dir", "", "Base directory for WebSocket scripts")
	devConsoleFlag := flag.Bool("devconsole", false, "Enable development console")

	flag.Parse()

	mainConfig.Addr = fmt.Sprintf("%s:%d", *addressFlag, *portFlag)
	mainConfig.BasePath = *basePathFlag

	switch *logLevelFlag {
	case "debug":
		mainConfig.LogLevel = libwebsocketd.LogDebug
		break
	case "trace":
		mainConfig.LogLevel = libwebsocketd.LogTrace
		break
	case "access":
		mainConfig.LogLevel = libwebsocketd.LogAccess
		break
	case "info":
		mainConfig.LogLevel = libwebsocketd.LogInfo
		break
	case "error":
		mainConfig.LogLevel = libwebsocketd.LogError
		break
	case "fatal":
		mainConfig.LogLevel = libwebsocketd.LogFatal
		break
	default:
		PrintHelp()
		os.Exit(1)
	}

	config.ReverseLookup = *reverseLookupFlag
	config.ScriptDir = *scriptDirFlag
	config.DevConsole = *devConsoleFlag
	config.StartupTime = time.Now()
	config.ServerSoftware = fmt.Sprintf("websocketd/%s", Version())

	if len(os.Args) == 1 {
		PrintHelp()
		os.Exit(1)
	}

	if *versionFlag {
		fmt.Printf("%s %s\n", filepath.Base(os.Args[0]), Version())
		os.Exit(2)
	}

	if *licenseFlag {
		fmt.Printf("%s %s\n", filepath.Base(os.Args[0]), Version())
		fmt.Printf("%s\n", libwebsocketd.License)
		os.Exit(2)
	}

	args := flag.Args()
	if len(args) < 1 && config.ScriptDir == "" {
		fmt.Fprintf(os.Stderr, "Please specify COMMAND or provide --dir argument.\n")
		os.Exit(1)
	}

	if len(args) > 0 {
		if config.ScriptDir != "" {
			fmt.Fprintf(os.Stderr, "Ambiguous. Provided COMMAND and --dir argument. Please only specify just one.\n")
			os.Exit(1)
		}
		config.CommandName = args[0]
		config.CommandArgs = flag.Args()[1:]
		config.UsingScriptDir = false
	}

	if len(config.ScriptDir) > 0 {
		scriptDir, err := filepath.Abs(config.ScriptDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not resolve absolute path to dir '%s'.\n", config.ScriptDir)
			os.Exit(1)
		}
		config.ScriptDir = scriptDir
		config.UsingScriptDir = true
	}

	mainConfig.Config = &config

	return mainConfig
}
