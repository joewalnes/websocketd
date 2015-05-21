// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/joewalnes/websocketd/libwebsocketd"
)

type Config struct {
	Addr              []string // TCP addresses to listen on. e.g. ":1234", "1.2.3.4:1234" or "[::1]:1234"
	MaxForks          int      // Number of allowable concurrent forks
	LogLevel          libwebsocketd.LogLevel
	CertFile, KeyFile string
	*libwebsocketd.Config
}

type Arglist []string

func (al *Arglist) String() string {
	return fmt.Sprintf("%v", []string(*al))
}

func (al *Arglist) Set(value string) error {
	*al = append(*al, value)
	return nil
}

// Borrowed from net/http/cgi
var defaultPassEnv = map[string]string{
	"darwin":  "PATH,DYLD_LIBRARY_PATH",
	"freebsd": "PATH,LD_LIBRARY_PATH",
	"hpux":    "PATH,LD_LIBRARY_PATH,SHLIB_PATH",
	"irix":    "PATH,LD_LIBRARY_PATH,LD_LIBRARYN32_PATH,LD_LIBRARY64_PATH",
	"linux":   "PATH,LD_LIBRARY_PATH",
	"openbsd": "PATH,LD_LIBRARY_PATH",
	"solaris": "PATH,LD_LIBRARY_PATH,LD_LIBRARY_PATH_32,LD_LIBRARY_PATH_64",
	"windows": "PATH,SystemRoot,COMSPEC,PATHEXT,WINDIR",
}

func parseCommandLine() *Config {
	var mainConfig Config
	var config libwebsocketd.Config

	flag.Usage = func() {}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// If adding new command line options, also update the help text in help.go.
	// The flag library's auto-generate help message isn't pretty enough.

	addrlist := Arglist(make([]string, 0, 1)) // pre-reserve for 1 address
	flag.Var(&addrlist, "address", "Interfaces to bind to (e.g. 127.0.0.1 or [::1]).")

	// server config options
	portFlag := flag.Int("port", 0, "HTTP port to listen on")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	licenseFlag := flag.Bool("license", false, "Print license and exit")
	logLevelFlag := flag.String("loglevel", "access", "Log level, one of: debug, trace, access, info, error, fatal")
	sslFlag := flag.Bool("ssl", false, "Use TLS on listening socket (see also --sslcert and --sslkey)")
	sslCert := flag.String("sslcert", "", "Should point to certificate PEM file when --ssl is used")
	sslKey := flag.String("sslkey", "", "Should point to certificate private key file when --ssl is used")
	maxForksFlag := flag.Int("maxforks", 0, "Max forks, zero means unlimited")

	// lib config options
	reverseLookupFlag := flag.Bool("reverselookup", true, "Perform reverse DNS lookups on remote clients")
	scriptDirFlag := flag.String("dir", "", "Base directory for WebSocket scripts")
	staticDirFlag := flag.String("staticdir", "", "Serve static content from this directory over HTTP")
	cgiDirFlag := flag.String("cgidir", "", "Serve CGI scripts from this directory over HTTP")
	devConsoleFlag := flag.Bool("devconsole", false, "Enable development console (cannot be used in conjunction with --staticdir)")
	passEnvFlag := flag.String("passenv", defaultPassEnv[runtime.GOOS], "List of envvars to pass to subprocesses (others will be cleaned out)")
	sameOriginFlag := flag.Bool("sameorigin", false, "Restrict upgrades if origin and host headers differ")
	allowOriginsFlag := flag.String("origin", "", "Restrict upgrades if origin does not match the list")

	headers := Arglist(make([]string, 0))
	headersWs := Arglist(make([]string, 0))
	headersHttp := Arglist(make([]string, 0))
	flag.Var(&headers, "header", "Custom headers for any response.")
	flag.Var(&headersWs, "header-ws", "Custom headers for successful WebSocket upgrade responses.")
	flag.Var(&headersHttp, "header-http", "Custom headers for all but WebSocket upgrade HTTP responses.")

	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			PrintHelp()
			os.Exit(0)
		} else {
			ShortHelp()
			os.Exit(2)
		}
	}

	port := *portFlag
	if port == 0 {
		if *sslFlag {
			port = 443
		} else {
			port = 80
		}
	}

	if socknum := len(addrlist); socknum != 0 {
		mainConfig.Addr = make([]string, socknum)
		for i, addrSingle := range addrlist {
			mainConfig.Addr[i] = fmt.Sprintf("%s:%d", addrSingle, port)
		}
	} else {
		mainConfig.Addr = []string{fmt.Sprintf(":%d", port)}
	}
	mainConfig.MaxForks = *maxForksFlag
	mainConfig.LogLevel = libwebsocketd.LevelFromString(*logLevelFlag)
	if mainConfig.LogLevel == libwebsocketd.LogUnknown {
		fmt.Printf("Incorrect loglevel flag '%s'. Use --help to see allowed values.\n", *logLevelFlag)
		ShortHelp()
		os.Exit(1)
	}

	config.Headers = []string(headers)
	config.HeadersWs = []string(headersWs)
	config.HeadersHTTP = []string(headersHttp)

	config.ReverseLookup = *reverseLookupFlag
	config.Ssl = *sslFlag
	config.ScriptDir = *scriptDirFlag
	config.StaticDir = *staticDirFlag
	config.CgiDir = *cgiDirFlag
	config.DevConsole = *devConsoleFlag
	config.StartupTime = time.Now()
	config.ServerSoftware = fmt.Sprintf("websocketd/%s", Version())

	if len(os.Args) == 1 {
		fmt.Printf("Command line arguments are missing.\n")
		ShortHelp()
		os.Exit(1)
	}

	if *versionFlag {
		fmt.Printf("%s %s\n", HelpProcessName(), Version())
		os.Exit(0)
	}

	if *licenseFlag {
		fmt.Printf("%s %s\n", HelpProcessName(), Version())
		fmt.Printf("%s\n", libwebsocketd.License)
		os.Exit(0)
	}

	// Reading SSL options
	if config.Ssl {
		if *sslCert == "" || *sslKey == "" {
			fmt.Fprintf(os.Stderr, "Please specify both --sslcert and --sslkey when requesting --ssl.\n")
			os.Exit(1)
		}
	} else {
		if *sslCert != "" || *sslKey != "" {
			fmt.Fprintf(os.Stderr, "You should not be using --ssl* flags when there is no --ssl option.\n")
			os.Exit(1)
		}
	}

	mainConfig.CertFile = *sslCert
	mainConfig.KeyFile = *sslKey

	// Building config.ParentEnv to avoid calling Environ all the time in the scripts
	// (caller is responsible for wiping environment if desired)
	config.ParentEnv = make([]string, 0)
	newlineCleaner := strings.NewReplacer("\n", " ", "\r", " ")
	for _, key := range strings.Split(*passEnvFlag, ",") {
		if key != "HTTPS" {
			if v := os.Getenv(key); v != "" {
				// inevitably adding flavor of libwebsocketd appendEnv func.
				// it's slightly nicer than in net/http/cgi implementation
				if clean := strings.TrimSpace(newlineCleaner.Replace(v)); clean != "" {
					config.ParentEnv = append(config.ParentEnv, fmt.Sprintf("%s=%s", key, clean))
				}
			}
		}
	}

	if *allowOriginsFlag != "" {
		config.AllowOrigins = strings.Split(*allowOriginsFlag, ",")
	}
	config.SameOrigin = *sameOriginFlag

	args := flag.Args()
	if len(args) < 1 && config.ScriptDir == "" && config.StaticDir == "" && config.CgiDir == "" {
		fmt.Fprintf(os.Stderr, "Please specify COMMAND or provide --dir, --staticdir or --cgidir argument.\n")
		ShortHelp()
		os.Exit(1)
	}

	if len(args) > 0 {
		if config.ScriptDir != "" {
			fmt.Fprintf(os.Stderr, "Ambiguous. Provided COMMAND and --dir argument. Please only specify just one.\n")
			ShortHelp()
			os.Exit(1)
		}
		if path, err := exec.LookPath(args[0]); err == nil {
			config.CommandName = path // This can be command in PATH that we are able to execute
			config.CommandArgs = flag.Args()[1:]
			config.UsingScriptDir = false
		} else {
			fmt.Fprintf(os.Stderr, "Unable to locate specified COMMAND '%s' in OS path.\n", args[0])
			ShortHelp()
			os.Exit(1)
		}
	}

	if config.ScriptDir != "" {
		scriptDir, err := filepath.Abs(config.ScriptDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not resolve absolute path to dir '%s'.\n", config.ScriptDir)
			ShortHelp()
			os.Exit(1)
		}
		inf, err := os.Stat(scriptDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not find your script dir '%s'.\n", config.ScriptDir)
			ShortHelp()
			os.Exit(1)
		}
		if !inf.IsDir() {
			fmt.Fprintf(os.Stderr, "Did you mean to specify COMMAND instead of --dir '%s'?\n", config.ScriptDir)
			ShortHelp()
			os.Exit(1)
		} else {
			config.ScriptDir = scriptDir
			config.UsingScriptDir = true
		}
	}

	if config.CgiDir != "" {
		if inf, err := os.Stat(config.CgiDir); err != nil || !inf.IsDir() {
			fmt.Fprintf(os.Stderr, "Your CGI dir '%s' is not pointing to an accessible directory.\n", config.CgiDir)
			ShortHelp()
			os.Exit(1)
		}
	}

	if config.StaticDir != "" {
		if inf, err := os.Stat(config.StaticDir); err != nil || !inf.IsDir() {
			fmt.Fprintf(os.Stderr, "Your static dir '%s' is not pointing to an accessible directory.\n", config.StaticDir)
			ShortHelp()
			os.Exit(1)
		}
	}

	mainConfig.Config = &config

	return &mainConfig
}
