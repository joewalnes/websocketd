// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

type Endpoint interface {
	StartReading()
	Terminate()
	Output() chan []byte
	Send([]byte) bool
}

// PipeEndpoints connects two endpoints bidirectionally. Each direction
// runs in its own goroutine so a blocking Send in one direction does not
// stall the other. This prevents deadlocks when both the process's stdin
// and stdout pipes are full simultaneously (e.g., large binary payloads).
//
// Backpressure is natural: if a Send blocks (e.g., pipe buffer full or
// slow WebSocket client), the corresponding direction stalls, which stops
// draining that endpoint's Output channel, which eventually blocks the
// producer. No unbounded buffering occurs.
func PipeEndpoints(e1, e2 Endpoint) {
	e1.StartReading()
	e2.StartReading()

	done := make(chan struct{}, 2)

	// e1 → e2 (e.g., WebSocket messages → process stdin)
	go func() {
		for msg := range e1.Output() {
			if !e2.Send(msg) {
				break
			}
		}
		done <- struct{}{}
	}()

	// e2 → e1 (e.g., process stdout → WebSocket messages)
	go func() {
		for msg := range e2.Output() {
			if !e1.Send(msg) {
				break
			}
		}
		done <- struct{}{}
	}()

	// Wait for either direction to finish (channel closed or send failed),
	// then terminate both endpoints to clean up the other direction.
	<-done
	e1.Terminate()
	e2.Terminate()
	<-done // wait for the second goroutine to finish
}
