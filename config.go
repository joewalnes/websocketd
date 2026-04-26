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
	RedirPort         int
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

// resolveAddresses builds the list of TCP addresses to listen on.
func resolveAddresses(addrlist []string, port int) []string {
	if len(addrlist) > 0 {
		addrs := make([]string, len(addrlist))
		for i, addr := range addrlist {
			addrs[i] = fmt.Sprintf("%s:%d", addr, port)
		}
		return addrs
	}
	return []string{fmt.Sprintf(":%d", port)}
}

// resolvePort determines the listening port, using defaults for HTTP (80) or HTTPS (443).
func resolvePort(portFlag int, ssl bool) int {
	if portFlag != 0 {
		return portFlag
	}
	if ssl {
		return 443
	}
	return 80
}

// validateSSL checks that SSL-related flags are consistent.
func validateSSL(ssl bool, certFile, keyFile string) error {
	if ssl {
		if certFile == "" || keyFile == "" {
			return fmt.Errorf("please specify both --sslcert and --sslkey when requesting --ssl")
		}
	} else {
		if certFile != "" || keyFile != "" {
			return fmt.Errorf("you should not be using --ssl* flags when there is no --ssl option")
		}
	}
	return nil
}

// buildParentEnv constructs the filtered parent environment variable list.
func buildParentEnv(passenv string) []string {
	env := make([]string, 0)
	newlineCleaner := strings.NewReplacer("\n", " ", "\r", " ")
	for _, key := range strings.Split(passenv, ",") {
		if key == "HTTPS" {
			continue
		}
		if v := os.Getenv(key); v != "" {
			if clean := strings.TrimSpace(newlineCleaner.Replace(v)); clean != "" {
				env = append(env, fmt.Sprintf("%s=%s", key, clean))
			}
		}
	}
	return env
}

// resolveCommand validates and resolves the command to execute.
// Returns the resolved command path, arguments, and whether it uses a script directory.
func resolveCommand(args []string, scriptDir string) (commandName string, commandArgs []string, usingScriptDir bool, err error) {
	if len(args) > 0 {
		if scriptDir != "" {
			return "", nil, false, fmt.Errorf("ambiguous: provided COMMAND and --dir argument, please only specify one")
		}
		path, lookErr := exec.LookPath(args[0])
		if lookErr != nil {
			return "", nil, false, fmt.Errorf("unable to locate specified COMMAND '%s' in OS path", args[0])
		}
		return path, args[1:], false, nil
	}
	return "", nil, false, nil
}

// resolveScriptDir validates and resolves the script directory path.
func resolveScriptDir(dir string) (string, error) {
	if dir == "" {
		return "", nil
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("could not resolve absolute path to dir '%s'", dir)
	}
	inf, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("could not find your script dir '%s'", dir)
	}
	if !inf.IsDir() {
		return "", fmt.Errorf("did you mean to specify COMMAND instead of --dir '%s'?", dir)
	}
	return absDir, nil
}

// validateDir checks that a directory path exists and is a directory.
func validateDir(dir, label string) error {
	if dir == "" {
		return nil
	}
	inf, err := os.Stat(dir)
	if err != nil || !inf.IsDir() {
		return fmt.Errorf("your %s '%s' is not pointing to an accessible directory", label, dir)
	}
	return nil
}

func parseCommandLine() *Config {
	var mainConfig Config
	var config libwebsocketd.Config

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.Usage = func() {}

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
	closeMsFlag := flag.Uint("closems", 0, "Time to start sending signals (0 never)")
	redirPortFlag := flag.Int("redirport", 0, "HTTP port to redirect to canonical --port address")

	// lib config options
	binaryFlag := flag.Bool("binary", false, "Set websocketd to experimental binary mode (default is line by line)")
	reverseLookupFlag := flag.Bool("reverselookup", false, "Perform reverse DNS lookups on remote clients")
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

	// Resolve port and addresses
	port := resolvePort(*portFlag, *sslFlag)
	mainConfig.Addr = resolveAddresses([]string(addrlist), port)
	mainConfig.MaxForks = *maxForksFlag
	mainConfig.RedirPort = *redirPortFlag

	// Validate log level
	mainConfig.LogLevel = libwebsocketd.LevelFromString(*logLevelFlag)
	if mainConfig.LogLevel == libwebsocketd.LogUnknown {
		fmt.Printf("Incorrect loglevel flag '%s'. Use --help to see allowed values.\n", *logLevelFlag)
		ShortHelp()
		os.Exit(1)
	}

	// Validate SSL
	if err := validateSSL(*sslFlag, *sslCert, *sslKey); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	mainConfig.CertFile = *sslCert
	mainConfig.KeyFile = *sslKey

	// Build lib config
	config.Headers = []string(headers)
	config.HeadersWs = []string(headersWs)
	config.HeadersHTTP = []string(headersHttp)
	config.CloseMs = *closeMsFlag
	config.Binary = *binaryFlag
	config.ReverseLookup = *reverseLookupFlag
	config.Ssl = *sslFlag
	config.ScriptDir = *scriptDirFlag
	config.StaticDir = *staticDirFlag
	config.CgiDir = *cgiDirFlag
	config.DevConsole = *devConsoleFlag
	config.StartupTime = time.Now()
	config.ServerSoftware = fmt.Sprintf("websocketd/%s", Version())
	config.HandshakeTimeout = time.Millisecond * 1500

	// Build parent environment
	config.ParentEnv = buildParentEnv(*passEnvFlag)

	// Parse origins
	if *allowOriginsFlag != "" {
		config.AllowOrigins = strings.Split(*allowOriginsFlag, ",")
	}
	config.SameOrigin = *sameOriginFlag

	// Resolve command or script directory
	args := flag.Args()
	if len(args) < 1 && config.ScriptDir == "" && config.StaticDir == "" && config.CgiDir == "" {
		fmt.Fprintf(os.Stderr, "Please specify COMMAND or provide --dir, --staticdir or --cgidir argument.\n")
		ShortHelp()
		os.Exit(1)
	}

	if len(args) > 0 {
		commandName, commandArgs, _, err := resolveCommand(args, config.ScriptDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			ShortHelp()
			os.Exit(1)
		}
		config.CommandName = commandName
		config.CommandArgs = commandArgs
		config.UsingScriptDir = false
	}

	if config.ScriptDir != "" {
		scriptDir, err := resolveScriptDir(config.ScriptDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			ShortHelp()
			os.Exit(1)
		}
		config.ScriptDir = scriptDir
		config.UsingScriptDir = true
	}

	if err := validateDir(config.CgiDir, "CGI dir"); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		ShortHelp()
		os.Exit(1)
	}

	if err := validateDir(config.StaticDir, "static dir"); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		ShortHelp()
		os.Exit(1)
	}

	mainConfig.Config = &config
	return &mainConfig
}
