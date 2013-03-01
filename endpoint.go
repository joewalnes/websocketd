// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type Endpoint interface {
	Terminate()
	Output() chan string
	Send(msg string) bool
}
