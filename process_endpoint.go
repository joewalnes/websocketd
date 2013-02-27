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

func readProcess(stdout io.ReadCloser, outbound chan<- string, done chan<- bool, config *Config) {
	bufstdout := bufio.NewReader(stdout)
	for {
		str, err := bufstdout.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Fatal("Unexpected read from process: ", err)
			} else {
				if config.Verbose {
					log.Print("process: CLOSED")
				}
			}
			break
		}
		msg := str[0 : len(str)-1] // Trim new line
		if config.Verbose {
			log.Print("process: OUT : <", msg, ">")
		}
		outbound <- msg
	}
	close(outbound)
	done <- true
}

func writeProcess(stdin io.WriteCloser, inbound <-chan string, done chan<- bool, config *Config) {
	bufstdin := bufio.NewWriter(stdin)
	for msg := range inbound {
		bufstdin.WriteString(msg)
		bufstdin.WriteString("\n")
		bufstdin.Flush()
	}
	done <- true
}

func pipeStdErr(stderr io.ReadCloser, done chan<- bool, config *Config) {
	bufstderr := bufio.NewReader(stderr)
	for {
		str, err := bufstderr.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Fatal("Unexpected read from process: ", err)
			} else {
				if config.Verbose {
					log.Print("process: CLOSED")
				}
			}
			break
		}
		msg := str[0 : len(str)-1] // Trim new line
		log.Print("process: STDERR : ", msg)
	}
	done <- true
}
