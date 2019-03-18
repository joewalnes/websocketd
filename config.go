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

type websocketdConfig struct {
	port              uint
	addr              []string // TCP addresses to listen on. e.g. ":1234", "1.2.3.4:1234" or "[::1]:1234"
	maxForks          int      // Number of allowable concurrent forks
	logLevelStr       string
	redirPort         int
	certFile, keyFile string
	passEnv           string

	service *libwebsocketd.Config

	version, license bool
}

// parse has odd signature but since it's internal func it should be okay... linter would not be happy
func parse(cnf *websocketdConfig, args []string) (error, []string) {
	websocketdFlags := flag.NewFlagSet("websocketd", flag.ContinueOnError)
	websocketdFlags.Usage = func() {}

	websocketdFlags.BoolVar(&cnf.version, "version", false, "Print version and exit")
	websocketdFlags.BoolVar(&cnf.license, "license", false, "Print license and exit")

	websocketdFlags.Var((*arglist)(&cnf.addr), "address", "Interfaces to bind to (e.g. 127.0.0.1 or [::1]).")
	websocketdFlags.UintVar(&cnf.port, "port", 0, "HTTP port to listen on")
	websocketdFlags.StringVar(&cnf.logLevelStr, "loglevel", "access", "Log level, one of: debug, trace, access, info, error, fatal")
	websocketdFlags.StringVar(&cnf.certFile, "sslcert", "", "Should point to certificate PEM file when --ssl is used")
	websocketdFlags.StringVar(&cnf.keyFile, "sslkey", "", "Should point to certificate private key file when --ssl is used")
	websocketdFlags.IntVar(&cnf.maxForks, "maxforks", 0, "Max forks, zero means unlimited")
	websocketdFlags.IntVar(&cnf.redirPort, "redirport", 0, "HTTP port to redirect to canonical --port address")
	websocketdFlags.StringVar(&cnf.passEnv, "passenv", defaultPassEnv[runtime.GOOS], "List of envvars to pass to subprocesses (others will be cleaned out)")

	// service-implementing flags
	websocketdFlags.BoolVar(&cnf.service.Ssl, "ssl", false, "Use TLS on listening socket (see also --sslcert and --sslkey)")
	websocketdFlags.UintVar(&cnf.service.CloseMs, "closems", 0, "Time to start sending signals (0 never)")
	websocketdFlags.BoolVar(&cnf.service.Binary, "binary", false, "Set websocketd to experimental binary mode (default is line by line)")
	websocketdFlags.BoolVar(&cnf.service.ReverseLookup, "reverselookup", false, "Perform reverse DNS lookups on remote clients")
	websocketdFlags.StringVar(&cnf.service.ScriptDir, "dir", "", "Base directory for WebSocket scripts")
	websocketdFlags.StringVar(&cnf.service.StaticDir, "staticdir", "", "Serve static content from this directory over HTTP")
	websocketdFlags.StringVar(&cnf.service.CgiDir, "cgidir", "", "Serve CGI scripts from this directory over HTTP")
	websocketdFlags.BoolVar(&cnf.service.DevConsole, "devconsole", false, "Enable development console (cannot be used in conjunction with --staticdir)")
	websocketdFlags.BoolVar(&cnf.service.SameOrigin, "sameorigin", false, "Restrict upgrades if origin and host headers differ")
	websocketdFlags.Var((*arglist)(&cnf.service.AllowOrigins), "origin", "Restrict upgrades if origin does not match the list")

	websocketdFlags.Var((*arglist)(&cnf.service.Headers), "header", "Custom headers for any response.")
	websocketdFlags.Var((*arglist)(&cnf.service.HeadersWs), "header-ws", "Custom headers for successful WebSocket upgrade responses.")
	websocketdFlags.Var((*arglist)(&cnf.service.HeadersHTTP), "header-http", "Custom headers for all but WebSocket upgrade HTTP responses.")

	return websocketdFlags.Parse(args), websocketdFlags.Args()
}

func parseCommandLine() (*websocketdConfig, error) {
	if len(os.Args) == 1 {
		return nil, fmt.Errorf("Command line arguments are missing.")
	}

	service := new(libwebsocketd.Config)
	config := &websocketdConfig{service: service}

	err, args := parse(config, os.Args[1:])
	if err != nil {
		return nil, err
	}

	if config.version || config.license {
		return config, nil
	}

	service.LogLevel = libwebsocketd.LevelFromString(config.logLevelStr)
	if service.LogLevel == libwebsocketd.LogUnknown {
		return nil, fmt.Errorf("Incorrect loglevel flag '%s'. Use --help to see allowed values.", config.logLevelStr)
	}

	// Reading SSL options
	if service.Ssl {
		if config.certFile == "" || config.keyFile == "" {
			return nil, fmt.Errorf("Please specify both --sslcert and --sslkey when requesting --ssl.")
		}
	} else {
		if config.certFile != "" || config.keyFile != "" {
			return nil, fmt.Errorf("You should not be using --ssl* flags when there is no --ssl option.")
		}
	}

	if config.port == 443 {
		if service.Ssl {
			config.port = 443
		} else {
			config.port = 80
		}
	}

	if socknum := len(config.addr); socknum != 0 {
		for i, addrSingle := range config.addr {
			if !strings.ContainsRune(addrSingle, ':') {
				config.addr[i] = fmt.Sprintf("%s:%d", addrSingle, config.port)
			}
		}
	} else {
		config.addr = []string{fmt.Sprintf(":%d", config.port)}
	}

	service.HandshakeTimeout = time.Millisecond * 1500 // only default for now

	// Building config.ParentEnv to avoid calling Environ all the time in the scripts
	// (caller is responsible for wiping environment if desired)
	service.ParentEnv = make([]string, 0)
	for _, key := range strings.Split(config.passEnv, ",") {
		if key != "HTTPS" {
			// even if var is empty, we pass it
			service.ParentEnv = append(service.ParentEnv, fmt.Sprintf("%s=%s", key, strings.TrimSpace(os.Getenv(key))))
		}
	}

	if ln := len(service.AllowOrigins); ln > 0 {
		// split values by comma if they are listed as groups
		tmp := make([]string, 0, ln)
		for _, orig := range service.AllowOrigins {
			pos := strings.IndexByte(orig, ',')
			for pos >= 0 {
				add, orig := strings.TrimSpace(orig[:pos]), orig[pos+1:]
				if len(add) > 0 {
					tmp = append(tmp, add)
				}
				pos = strings.IndexByte(orig, ',')
			}
			if orig = strings.TrimSpace(orig); len(orig) > 0 {
				tmp = append(tmp, orig)
			}
		}
		service.AllowOrigins = tmp
	}

	if len(args) < 1 && service.ScriptDir == "" && service.StaticDir == "" && service.CgiDir == "" {
		return nil, fmt.Errorf("Please specify COMMAND or provide --dir, --staticdir or --cgidir argument.")
	}

	if len(args) > 0 {
		if service.ScriptDir != "" {
			return nil, fmt.Errorf("Ambiguous. Provided COMMAND and --dir argument. Please only specify just one.")
		}
		if path, err := exec.LookPath(args[0]); err == nil {
			service.CommandName = path // This can be command in PATH that we are able to execute
			service.CommandArgs = flag.Args()[1:]
			service.UsingScriptDir = false
		} else {
			return nil, fmt.Errorf("Unable to locate specified COMMAND '%s' in OS path.", args[0])
		}
	}

	if service.ScriptDir != "" {
		scriptDir, err := filepath.Abs(service.ScriptDir)
		if err != nil {
			return nil, fmt.Errorf("Could not resolve absolute path to dir '%s'.", service.ScriptDir)
		}
		inf, err := os.Stat(scriptDir)
		if err != nil {
			return nil, fmt.Errorf("Could not find your script dir '%s'.", service.ScriptDir)
		}
		if !inf.IsDir() {
			return nil, fmt.Errorf("Did you mean to specify COMMAND instead of --dir '%s'?", service.ScriptDir)
		} else {
			service.ScriptDir = scriptDir
			service.UsingScriptDir = true
		}
	}

	if service.CgiDir != "" {
		if inf, err := os.Stat(service.CgiDir); err != nil || !inf.IsDir() {
			return nil, fmt.Errorf("Your CGI dir '%s' is not pointing to an accessible directory.", service.CgiDir)
		}
		if config.service.DevConsole {
			return nil, fmt.Errorf("Invalid parameters: --devconsole cannot be used with --cgidir. Pick one.")
		}
	}

	if service.StaticDir != "" {
		if inf, err := os.Stat(service.StaticDir); err != nil || !inf.IsDir() {
			return nil, fmt.Errorf("Your static dir '%s' is not pointing to an accessible directory.", service.StaticDir)
		}
		if config.service.DevConsole {
			return nil, fmt.Errorf("Invalid parameters: --devconsole cannot be used with --staticdir. Pick one.")
		}
	}

	return config, nil
}

type arglist []string

func (al *arglist) String() string {
	return fmt.Sprintf("%v", []string(*al))
}

func (al *arglist) Set(value string) error {
	*al = append(*al, value)
	return nil
}
