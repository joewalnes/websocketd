// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	help = `
{{binary}} ({{version}})

{{binary}} is a command line tool that will allow any executable program
that accepts input on stdin and produces output on stdout to be turned into
a WebSocket server.

Usage:

  Export a single executable program a WebSocket server:

    {{binary}} [options] program [program args]

Options:

  --port=PORT                    HTTP port to listen on.

  --address=ADDRESS              Address to bind to. 
                                 Default: 0.0.0.0 (all)

  --basepath=PATH                Base path in URLs to serve from. 
                                 Default: / (root of domain)

  --verbose                      Enable verbose logging to stdout.

  --reverselookup={true,false}   Perform DNS reverse lookups on remote clients.
                                 Default: true

Full documentation at http://websocketd.com/

Copyright 2013 Joe Walnes and the websocketd team. All rights reserved.
BSD license: https://raw.github.com/joewalnes/websocketd/master/LICENSE
`
)

func PrintHelp() {
	msg := strings.Trim(help, " \n")
	msg = strings.Replace(msg, "{{binary}}", filepath.Base(os.Args[0]), -1)
	msg = strings.Replace(msg, "{{version}}", Version(), -1)
	fmt.Fprintf(os.Stderr, "%s\n", msg)
}
