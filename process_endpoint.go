// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"io"
	"log"
)

type ProcessEndpoint struct {
	process    *LaunchedProcess
	bufferedIn *bufio.Writer
	output     chan string
}

func NewProcessEndpoint(process *LaunchedProcess) *ProcessEndpoint {
	return &ProcessEndpoint{
		process:    process,
		bufferedIn: bufio.NewWriter(process.stdin),
		output:     make(chan string)}
}

func (pe *ProcessEndpoint) Terminate() {
	pe.process.stdin.Close()

	err := pe.process.cmd.Process.Kill()
	if err != nil {
		log.Printf("websocketd failed to kill process %v", pe.process.cmd.Process.Pid)
	}

	err = pe.process.cmd.Wait()
	if err != nil {
		log.Printf("websocketd couldn't reap process %v", pe.process.cmd.Process.Pid)
	}
}

func (pe *ProcessEndpoint) Output() chan string {
	return pe.output
}

func (pe *ProcessEndpoint) Send(msg string) bool {
	pe.bufferedIn.WriteString(msg)
	pe.bufferedIn.WriteString("\n")
	pe.bufferedIn.Flush()
	return true
}

func (pe *ProcessEndpoint) ReadOutput(input io.ReadCloser, config *Config) {
	bufin := bufio.NewReader(input)
	for {
		str, err := bufin.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Fatalf("Unexpected while reading process stdout: ", err)
			} else {
				if config.Verbose {
					log.Printf("process stdout: CLOSED")
				}
			}
			break
		}
		msg := str[0 : len(str)-1] // Trim new line
		if config.Verbose {
			log.Printf("process: OUT : <%s>", msg)
		}
		pe.output <- msg
	}
	close(pe.output)
}


func (pe *ProcessEndpoint) pipeStdErr(config *Config) {
	bufstderr := bufio.NewReader(pe.process.stderr)
	for {
		str, err := bufstderr.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Fatal("Unexpected read from process: ", err)
			} else {
				if config.Verbose {
					log.Print("process stderr: CLOSED")
				}
			}
			break
		}
		msg := str[0 : len(str)-1] // Trim new line
		log.Print("process: STDERR : ", msg)
	}
}
