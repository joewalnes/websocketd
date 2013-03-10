// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"io"
)

type ProcessEndpoint struct {
	process    *LaunchedProcess
	bufferedIn *bufio.Writer
	output     chan string
	log        *LogScope
}

func NewProcessEndpoint(process *LaunchedProcess, log *LogScope) *ProcessEndpoint {
	return &ProcessEndpoint{
		process:    process,
		bufferedIn: bufio.NewWriter(process.stdin),
		output:     make(chan string),
		log:        log}
}

func (pe *ProcessEndpoint) Terminate() {
	pe.process.stdin.Close()

	err := pe.process.cmd.Process.Kill()
	if err != nil {
		pe.log.Debug("process", "Failed to kill process %v: %s", pe.process.cmd.Process.Pid, err)
	}

	pe.process.cmd.Wait()
	if err != nil {
		pe.log.Debug("process", "Failed to reap process %v: %s", pe.process.cmd.Process.Pid, err)
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
				pe.log.Error("process", "Unexpected STDOUT read from process: %s", err)
			} else {
				pe.log.Debug("process", "Process STDOUT closed")
			}
			break
		}
		pe.output <- trimEOL(str)
	}
	close(pe.output)
}

func (pe *ProcessEndpoint) pipeStdErr(config *Config) {
	bufstderr := bufio.NewReader(pe.process.stderr)
	for {
		str, err := bufstderr.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				pe.log.Error("process", "Unexpected STDERR read from process: %s", err)
			} else {
				pe.log.Debug("process", "Process STDERR closed")
			}
			break
		}
		pe.log.Error("stderr", "%s", trimEOL(str))
	}
}

func trimEOL(s string) string {
	// Handles unixy style \n and windowsy style \r\n
	trimCount := 0
	if len(s) > 0 && s[len(s)-1] == '\n' {
		trimCount = 1
		if len(s) > 1 && s[len(s)-2] == '\r' {
			trimCount = 2
		}
	}
	if trimCount == 0 {
		return s
	}
	return s[0 : len(s)-trimCount]
}
