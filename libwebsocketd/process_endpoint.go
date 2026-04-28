// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"syscall"
	"sync"
	"time"
)

type ProcessEndpoint struct {
	process    *LaunchedProcess
	closetime  time.Duration
	output     chan []byte
	log        *LogScope
	bin        bool
	passStderr bool
	wg         sync.WaitGroup
}

func NewProcessEndpoint(process *LaunchedProcess, bin bool, log *LogScope, passStderr bool) *ProcessEndpoint {
	return &ProcessEndpoint{
		process:    process,
		output:     make(chan []byte),
		log:        log,
		bin:        bin,
		passStderr: passStderr,
	}
}

func (pe *ProcessEndpoint) Terminate() {
	terminated := make(chan struct{})
	go func() {
		if err := pe.process.cmd.Wait(); err != nil {
			pe.log.Debug("process", "Process exit: %s", err)
		}
		terminated <- struct{}{}
	}()

	// for some processes this is enough to finish them...
	if err := pe.process.stdin.Close(); err != nil {
		pe.log.Debug("process", "STDIN close: %s", err)
	}

	pid := pe.process.cmd.Process.Pid

	// Escalating termination: stdin close → SIGINT → SIGTERM → SIGKILL
	signals := []struct {
		signal  os.Signal
		name    string
		timeout time.Duration
	}{
		{nil, "stdin was closed", 100*time.Millisecond + pe.closetime},
		{syscall.SIGINT, "SIGINT", 250*time.Millisecond + pe.closetime},
		{syscall.SIGTERM, "SIGTERM", 500*time.Millisecond + pe.closetime},
		{syscall.SIGKILL, "SIGKILL", 1000 * time.Millisecond},
	}

	for _, step := range signals {
		if step.signal != nil {
			if err := pe.process.cmd.Process.Signal(step.signal); err != nil {
				pe.log.Error("process", "%s unsuccessful to %v: %s", step.name, pid, err)
			}
		}
		select {
		case <-terminated:
			pe.log.Debug("process", "Process %v terminated after %s", pid, step.name)
			return
		case <-time.After(step.timeout):
		}
	}

	pe.log.Error("process", "SIGKILL did not terminate %v!", pid)
}

func (pe *ProcessEndpoint) Output() chan []byte {
	return pe.output
}

func (pe *ProcessEndpoint) Send(msg []byte) bool {
	_, err := pe.process.stdin.Write(msg)
	if err != nil {
		pe.log.Debug("process", "Cannot write to STDIN: %s", err)
		return false
	}
	return true
}

func (pe *ProcessEndpoint) StartReading() {
	if pe.passStderr {
		pe.wg.Add(2)
		go pe.readStderrTagged()
		go pe.readStdoutTagged()
		go pe.closeOutputWhenDone()
	} else {
		go pe.logStderr()
		if pe.bin {
			go pe.readBinaryOutput()
		} else {
			go pe.readTextOutput()
		}
	}
}

func (pe *ProcessEndpoint) closeOutputWhenDone() {
	pe.wg.Wait()
	close(pe.output)
}

func (pe *ProcessEndpoint) readTextOutput() {
	bufin := bufio.NewReader(pe.process.stdout)
	for {
		buf, err := bufin.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				pe.log.Error("process", "Unexpected error while reading STDOUT from process: %s", err)
			} else {
				pe.log.Debug("process", "Process STDOUT closed")
			}
			break
		}
		pe.output <- trimEOL(buf)
	}
	close(pe.output)
}

func (pe *ProcessEndpoint) readBinaryOutput() {
	buf := make([]byte, 10*1024*1024)
	for {
		n, err := pe.process.stdout.Read(buf)
		if err != nil {
			if err != io.EOF {
				pe.log.Error("process", "Unexpected error while reading STDOUT from process: %s", err)
			} else {
				pe.log.Debug("process", "Process STDOUT closed")
			}
			break
		}
		pe.output <- append(make([]byte, 0, n), buf[:n]...) // cloned buffer
	}
	close(pe.output)
}

func (pe *ProcessEndpoint) readStdoutTagged() {
	defer pe.wg.Done()
	bufin := bufio.NewReader(pe.process.stdout)
	for {
		buf, err := bufin.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				pe.log.Error("process", "Unexpected error while reading STDOUT from process: %s", err)
			} else {
				pe.log.Debug("process", "Process STDOUT closed")
			}
			break
		}
		pe.output <- tagMessage("stdout", string(trimEOL(buf)))
	}
}

func (pe *ProcessEndpoint) readStderrTagged() {
	defer pe.wg.Done()
	bufin := bufio.NewReader(pe.process.stderr)
	for {
		buf, err := bufin.ReadSlice('\n')
		if err != nil {
			if err != io.EOF {
				pe.log.Error("process", "Unexpected error while reading STDERR from process: %s", err)
			} else {
				pe.log.Debug("process", "Process STDERR closed")
			}
			break
		}
		pe.log.Error("stderr", "%s", string(trimEOL(buf)))
		pe.output <- tagMessage("stderr", string(trimEOL(buf)))
	}
}

func tagMessage(stream, data string) []byte {
	msg := fmt.Sprintf("{\"stream\":\"%s\",\"data\":%s}", stream, jsonQuote(data))
	return []byte(msg)
}

func jsonQuote(s string) string {
	b := make([]byte, 0, len(s)+2)
	b = append(b, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '"':
			b = append(b, '\\', c)
		case '\n':
			b = append(b, '\\', 'n')
		case '\r':
			b = append(b, '\\', 'r')
		case '\t':
			b = append(b, '\\', 't')
		default:
			b = append(b, c)
		}
	}
	b = append(b, '"')
	return string(b)
}

func (pe *ProcessEndpoint) logStderr() {
	bufstderr := bufio.NewReader(pe.process.stderr)
	for {
		buf, err := bufstderr.ReadSlice('\n')
		if err != nil {
			if err != io.EOF {
				pe.log.Error("process", "Unexpected error while reading STDERR from process: %s", err)
			} else {
				pe.log.Debug("process", "Process STDERR closed")
			}
			break
		}
		pe.log.Error("stderr", "%s", string(trimEOL(buf)))
	}
}

// trimEOL cuts unixy style \n and windowsy style \r\n suffix from the string
func trimEOL(b []byte) []byte {
	lns := len(b)
	if lns > 0 && b[lns-1] == '\n' {
		lns--
		if lns > 0 && b[lns-1] == '\r' {
			lns--
		}
	}
	return b[:lns]
}
