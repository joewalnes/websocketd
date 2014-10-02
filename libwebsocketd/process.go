// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"bufio"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var ErrNoConsumers = errors.New("All consumers are gone")
var ErrProcessFinished = errors.New("Process already finished")
var ErrUnknownConsumer = errors.New("No consumer to unsubscribe")

var RcvrTimeout = time.Second * 5
var StdoutBufSize int64 = 1024 * 1024
var StdoutBufLines int64 = 10000

// ExternalProcess holds info about running process and sends info to subscribers using channels
type ExternalProcess struct {
	cmd         *exec.Cmd
	in          io.WriteCloser
	mux         *sync.Mutex
	terminating chan int
	log         *LogScope
}

func (p *ExternalProcess) wait() {
	p.cmd.Wait()
	p.log.Debug("process", "Process completed, status: %s", p.cmd.ProcessState.String())
}

// LaunchProcess initializes ExternalProcess struct fields. Command pipes for standard input/output are established and first consumer channel is returned.
func LaunchProcess(cmd *exec.Cmd, log *LogScope) (*ExternalProcess, <-chan string, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Debug("process", "Unable to create p")
		return nil, nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, nil, err
	}

	p := &ExternalProcess{
		cmd:         cmd,
		in:          stdin,
		mux:         new(sync.Mutex),
		terminating: make(chan int),
		log:         log,
	}
	log.Associate("pid", strconv.Itoa(p.Pid()))
	p.log.Trace("process", "Command started, first consumer channel created")

	consumer := make(chan string)

	// Run output listeners
	go p.process_stdout(stdout, consumer)
	go p.process_stderr(stderr)

	return p, consumer, nil
}

// Terminate tries to stop process forcefully using interrupt and kill signals with a second of waiting time between them. If the kill is unsuccessful, it might be repeated
// again and again while system accepts those attempts.
func (p *ExternalProcess) Terminate() {
	// prevent double entrance to this subroutine...
	p.mux.Lock()
	defer p.mux.Unlock()

	if p.cmd.ProcessState == nil {
		go func() { p.wait(); close(p.terminating) }()

		select {
		case <-p.terminating:
			return
		case <-time.After(time.Millisecond * 10):
			p.log.Debug("process", "Sending SIGINT to %d", p.Pid())
			err := p.cmd.Process.Signal(os.Interrupt)
			if err != nil {
				p.log.Error("process", "could not send SIGINT %s", err)
			}
		}

		for {
			select {
			case <-p.terminating:
				return
			case <-time.After(time.Second):
				p.log.Error("process", "process did not react to SIGINT, sending SIGKILL")
				err := p.cmd.Process.Signal(os.Kill)
				if err != nil {
					p.log.Error("process", "could not send SIGKILL %s (process got lost?)", err)
					return
				}
			}
		}
	}
}

// Pid is a helper function to return Pid from OS Process
func (e *ExternalProcess) Pid() int {
	return e.cmd.Process.Pid
}

// PassInput delivers particular string to the process, involves locking input channel
func (p *ExternalProcess) PassInput(s string) error {
	if p.cmd.ProcessState != nil {
		return ErrProcessFinished
	}
	p.mux.Lock()
	defer p.mux.Unlock()
	_, err := io.WriteString(p.in, s+"\n")
	if err != nil {
		p.log.Info("process", "Unable to write string to a process: %s", err)
		return err
	}
	p.log.Debug("process", "Passed input string %#v", s)
	return nil
}

// process_stdout is a background function that reads from process and muxes output to all subscribed channels
func (p *ExternalProcess) process_stdout(r io.ReadCloser, c chan string) {
	bsize, backlog := int64(0), make(chan string, StdoutBufLines)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
	LOOP:
		for s := range backlog {
			select {
			case c <- s:
				l := len(s)
				p.log.Trace("process", "Sent %d bytes to websocket handler", l)
				atomic.AddInt64(&bsize, int64(-l))
			case <-p.terminating:
				p.log.Trace("process", "Websocket handler connection was terminated...")
				break LOOP
			case <-time.After(RcvrTimeout):
				p.log.Trace("process", "Websocket handler timed out with %d messages in queue (%d bytes), terminating...", len(backlog), bsize)
				r.Close()
				break LOOP
			}
		}
		close(c)
		wg.Done()
	}()

	buf := bufio.NewReader(r)
	for {
		str, err := buf.ReadString('\n')
		if str != "" {
			str = trimEOL(str)
			backlog <- str
			if sz := atomic.AddInt64(&bsize, int64(len(str))); sz > StdoutBufSize {
				p.log.Trace("process", "Websocket handler did not process %d messages (%d bytes), terminating...", len(backlog), bsize)
				break
			}
		}
		if err != nil {
			p.log.Debug("process", "STDOUT stream ended: %s", err)
			break
		}
	}
	close(backlog)
	wg.Wait()
	p.Terminate()
}

// process_stderr is a function to log process output to STDERR
func (p *ExternalProcess) process_stderr(r io.ReadCloser) {
	buf := bufio.NewReader(r)
	for {
		str, err := buf.ReadString('\n')
		str = trimEOL(str)
		if str != "" {
			p.log.Error("stderr", "%s", str)
		}
		if err != nil {
			p.log.Debug("process", "STDERR stream ended: %s", err)
			break
		}
	}
}

// trimEOL cuts unixy style \n and windowsy style \r\n suffix from the string
func trimEOL(s string) string {
	lns := len(s)
	if lns > 0 && s[lns-1] == '\n' {
		lns--
		if lns > 0 && s[lns-1] == '\r' {
			lns--
		}
	}
	return s[0:lns]
}
