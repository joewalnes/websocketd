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
	"strconv"
	"strings"
	"time"

	"github.com/joewalnes/websocketd/internal/cliflags"
	"github.com/joewalnes/websocketd/libwebsocketd"
)

type Config struct {
	Addr              []string    // TCP addresses to listen on. e.g. ":1234", "1.2.3.4:1234" or "[::1]:1234"
	UnixSocket        string      // Path of a Unix domain socket to listen on, in addition to (or instead of) Addr
	SocketMode        os.FileMode // Permission bits to force on the Unix socket file (0 = follow umask)
	MaxForks          int         // Number of allowable concurrent forks
	LogLevel          libwebsocketd.LogLevel
	RedirPort         int
	CertFile, KeyFile string
	*libwebsocketd.Config
}

// schemelessOriginWarnings returns the --origin entries that carry no scheme
// and therefore also match insecure http origins. It only reports when the
// server itself runs with TLS (--ssl), the case where accepting an http origin
// is usually unintended; the operator can prefix "https://" to require TLS.
func schemelessOriginWarnings(ssl bool, allowOrigins []string) []string {
	if !ssl {
		return nil
	}
	var out []string
	for _, o := range allowOrigins {
		if !strings.Contains(o, "://") {
			out = append(out, o)
		}
	}
	return out
}

// parseSocketMode parses the --socketmode flag: an octal permission mode
// such as "0700". The empty string means "not set" and leaves the socket
// file to the process umask; an explicit zero is rejected because it would
// make the socket unusable for everyone, owner included.
func parseSocketMode(s string) (os.FileMode, error) {
	if s == "" {
		return 0, nil
	}
	mode, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("--socketmode %q is not an octal permission mode (e.g. 0700)", s)
	}
	if mode > 0o777 {
		return 0, fmt.Errorf("--socketmode %q has bits beyond permission bits (keep it within 0777)", s)
	}
	if mode == 0 {
		return 0, fmt.Errorf("--socketmode 0 would make the socket unusable; pick a mode like 0700")
	}
	return os.FileMode(mode), nil
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

// wantsUnixSocketOnly reports whether the user asked to serve exclusively
// over a Unix domain socket, with no TCP listener at all. This holds only
// when --unixsocket is given and nothing else implies a TCP listener is
// wanted (--port, --address, or --redirport); otherwise the Unix socket
// (if any) is served alongside the usual TCP listener(s).
func wantsUnixSocketOnly(unixSocket string, portFlag int, addrlist []string, redirPort int) bool {
	return unixSocket != "" && portFlag == 0 && len(addrlist) == 0 && redirPort == 0
}

// validateSSL checks that SSL-related flags are consistent. In particular,
// --sslca requires --ssl: --sslca enables mutual TLS (verifying a client
// certificate during the handshake), which only exists when the listener
// itself is running TLS. Without --ssl there is no handshake to verify a
// client certificate in, so --sslca alone used to be accepted and silently
// have no effect, serving plain HTTP with no client verification (issue
// #477).
func validateSSL(ssl bool, certFile, keyFile, caFile string) error {
	if ssl {
		if certFile == "" || keyFile == "" {
			return fmt.Errorf("please specify both --sslcert and --sslkey when requesting --ssl")
		}
	} else {
		if certFile != "" || keyFile != "" {
			return fmt.Errorf("you should not be using --ssl* flags when there is no --ssl option")
		}
		if caFile != "" {
			return fmt.Errorf("--sslca requires --ssl (mutual TLS has no effect without a TLS listener); add --ssl with --sslcert and --sslkey, or drop --sslca")
		}
	}
	return nil
}

// validateBinaryPassStderr checks that --binary and --passstderr aren't both
// set. Tagging binary chunks as JSON isn't implemented (--passstderr always
// reads line by line), so combining the two would silently discard --binary
// instead of behaving as either flag alone.
func validateBinaryPassStderr(binary bool, passStderr bool) error {
	if binary && passStderr {
		return fmt.Errorf("please only specify one of --binary and --passstderr")
	}
	return nil
}

// make sure binary and raw are not used together
func validateRawBinary(raw bool, binary bool) error {
	if binary && raw {
		return fmt.Errorf("please only specify one of --binary and --raw")
	}
	return nil
}

// validateAnyOrigin checks that --anyorigin is not combined with an actual
// origin policy. The flags say opposite things, and silently preferring one
// would hide operator confusion.
func validateAnyOrigin(anyOrigin, sameOrigin bool, allowOrigins []string) error {
	if anyOrigin && (sameOrigin || allowOrigins != nil) {
		return fmt.Errorf("--anyorigin means 'accept any origin' and cannot be combined with --sameorigin or --origin, which restrict it")
	}
	return nil
}

// validateMaxFrameSize rejects negative --maxframesize values. The read
// limit is only applied for positive values, so a negative value silently
// meant "unlimited" — the one value an operator can pass that quietly
// removes the DoS protection the flag exists for (issue #472).
func validateMaxFrameSize(maxFrameSize int64) error {
	if maxFrameSize < 0 {
		return fmt.Errorf("--maxframesize must not be negative; use 0 for unlimited")
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
// Returns the resolved command path and arguments.
func resolveCommand(args []string, scriptDir string) (commandName string, commandArgs []string, err error) {
	if len(args) > 0 {
		if scriptDir != "" {
			return "", nil, fmt.Errorf("ambiguous: provided COMMAND and --dir argument, please only specify one")
		}
		path, lookErr := exec.LookPath(args[0])
		if lookErr != nil {
			return "", nil, fmt.Errorf("unable to locate specified COMMAND '%s' in OS path", args[0])
		}
		return path, args[1:], nil
	}
	return "", nil, nil
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

// exitWithError prints err to stderr and exits with status 1. It is the
// common tail of every parseCommandLine validation: report and stop.
func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}

// exitWithUsageError is exitWithError plus the short usage summary, for
// validation failures where reminding the operator of the flags helps.
func exitWithUsageError(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	ShortHelp()
	os.Exit(1)
}

func parseCommandLine() *Config {
	var mainConfig Config
	var config libwebsocketd.Config

	fv := cliflags.Register()
	flag.CommandLine = fv.FS

	err := fv.FS.Parse(os.Args[1:])
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

	if *fv.Version {
		fmt.Printf("%s %s\n", HelpProcessName(), Version())
		os.Exit(0)
	}

	if *fv.License {
		fmt.Printf("%s %s\n", HelpProcessName(), Version())
		fmt.Printf("%s\n", libwebsocketd.License)
		os.Exit(0)
	}

	// Resolve port and addresses. A bare --unixsocket with no --port,
	// --address, or --redirport means Unix-socket-only: skip the default
	// TCP listener entirely rather than also binding ":80".
	if !wantsUnixSocketOnly(*fv.UnixSocket, *fv.Port, []string(fv.Addrlist), *fv.RedirPort) {
		port := resolvePort(*fv.Port, *fv.SSL)
		mainConfig.Addr = resolveAddresses([]string(fv.Addrlist), port)
	}
	mainConfig.UnixSocket = *fv.UnixSocket
	socketMode, err := parseSocketMode(*fv.SocketMode)
	if err != nil {
		exitWithError(err)
	}
	mainConfig.SocketMode = socketMode
	mainConfig.MaxForks = *fv.MaxForks
	mainConfig.RedirPort = *fv.RedirPort

	// Validate log level
	mainConfig.LogLevel = libwebsocketd.LevelFromString(*fv.LogLevel)
	if mainConfig.LogLevel == libwebsocketd.LogUnknown {
		fmt.Printf("Incorrect loglevel flag '%s'. Use --help to see allowed values.\n", *fv.LogLevel)
		ShortHelp()
		os.Exit(1)
	}

	// Validate SSL
	if err := validateSSL(*fv.SSL, *fv.SSLCert, *fv.SSLKey, *fv.SSLCA); err != nil {
		exitWithError(err)
	}
	mainConfig.CertFile = *fv.SSLCert
	mainConfig.KeyFile = *fv.SSLKey

	// Validate --binary / --passstderr
	if err := validateBinaryPassStderr(*fv.Binary, *fv.PassStderr); err != nil {
		exitWithError(err)
	}

	// Validate --binary / --raw
	if err := validateRawBinary(*fv.Raw, *fv.Binary); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	// Validate --maxframesize
	if err := validateMaxFrameSize(*fv.MaxFrameSize); err != nil {
		exitWithError(err)
	}

	// Build lib config
	config.Headers = []string(fv.Headers)
	config.HeadersWs = []string(fv.HeadersWs)
	config.HeadersHTTP = []string(fv.HeadersHttp)
	config.CloseMs = *fv.CloseMs
	config.PingInterval = time.Duration(*fv.PingMs) * time.Millisecond
	config.MaxFrameSize = *fv.MaxFrameSize
	config.Binary = *fv.Binary
	config.Raw = *fv.Raw
	config.PassStderr = *fv.PassStderr
	config.ReverseLookup = *fv.ReverseLookup
	config.Ssl = *fv.SSL
	config.SslCaFile = *fv.SSLCA
	config.ScriptDir = *fv.ScriptDir
	config.StaticDir = *fv.StaticDir
	config.CgiDir = *fv.CgiDir
	config.DevConsole = *fv.DevConsole
	config.StartupTime = time.Now()
	config.ServerSoftware = fmt.Sprintf("websocketd/%s", Version())
	config.HandshakeTimeout = time.Millisecond * 1500

	// Build parent environment
	config.ParentEnv = buildParentEnv(*fv.PassEnv)

	// Parse origins
	if *fv.AllowOrigins != "" {
		config.AllowOrigins = strings.Split(*fv.AllowOrigins, ",")
	}
	config.SameOrigin = *fv.SameOrigin
	config.AnyOrigin = *fv.AnyOrigin

	// Validate --anyorigin against actual origin policies
	if err := validateAnyOrigin(*fv.AnyOrigin, *fv.SameOrigin, config.AllowOrigins); err != nil {
		exitWithError(err)
	}

	// Resolve command or script directory
	args := fv.FS.Args()
	if len(args) < 1 && config.ScriptDir == "" && config.StaticDir == "" && config.CgiDir == "" {
		//lint:ignore ST1005 COMMAND is ok to be upper case
		exitWithUsageError(fmt.Errorf("Please specify COMMAND or provide --dir, --staticdir or --cgidir argument."))
	}

	if len(args) > 0 {
		commandName, commandArgs, err := resolveCommand(args, config.ScriptDir)
		if err != nil {
			exitWithUsageError(err)
		}
		config.CommandName = commandName
		config.CommandArgs = commandArgs
		config.UsingScriptDir = false
	}

	if config.ScriptDir != "" {
		scriptDir, err := resolveScriptDir(config.ScriptDir)
		if err != nil {
			exitWithUsageError(err)
		}
		config.ScriptDir = scriptDir
		config.UsingScriptDir = true
	}

	if err := validateDir(config.CgiDir, "CGI dir"); err != nil {
		exitWithUsageError(err)
	}

	if err := validateDir(config.StaticDir, "static dir"); err != nil {
		exitWithUsageError(err)
	}

	mainConfig.Config = &config
	return &mainConfig
}
