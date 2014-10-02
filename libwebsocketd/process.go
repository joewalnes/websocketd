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

// RcvrTimeout is a very short duration to determine if subscriber is unable to process data quickly enough.
// Zero is not practical because it would cause packet receiving to block while OS passes data via Pipe to process.
var RcvrTimeout = time.Second * 10

// StdoutBufSize is a size to limit max amount of data read from process and stored inside of Websocketd process
var StdoutBufSize = 10 * 1024 * 1024

// ExternalProcess holds info about running process and sends info to subscribers using channels
type ExternalProcess struct {
	cmd *exec.Cmd

	in    io.WriteCloser
	inmux *sync.Mutex

	terminating int32

	log *LogScope
}

func (p *ExternalProcess) wait() {
	atomic.StoreInt32(&p.terminating, 1)
	p.cmd.Wait()

	// if l := len(p.consumers); l > 0 {
	// 	p.log.Trace("process", "Closing %d consumer channels", l)
	// 	for _, x := range p.consumers {
	// 		close(x)
	// 	}
	// }
	// p.consumers = nil

	p.log.Debug("process", "Process completed, status: %s", p.cmd.ProcessState.String())
}

// LaunchProcess initializes ExternalProcess struct fields. Command pipes for standard input/output are established and first consumer channel is returned.
func LaunchProcess(cmd *exec.Cmd, log *LogScope) (*ExternalProcess, <-chan string, error) {
	// TODO: Investigate alternative approaches. exec.Cmd uses real OS pipes which spends new filehandler each.
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
		inmux:       new(sync.Mutex),
		terminating: 0,
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
	if p.cmd.ProcessState == nil {
		p.log.Debug("process", "Sending SIGINT to %d", p.Pid())

		// wait for process completion in background and report to channel
		term := make(chan int)
		go func() { p.wait(); close(term) }()

		err := p.cmd.Process.Signal(os.Interrupt)
		if err != nil {
			p.log.Error("process", "could not send SIGINT %s", err)
		}
		for {
			select {
			case <-term:
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

// Unubscribe signals back from the consumer and helps to finish process if output is quiet and all subscribers disconnected
func (p *ExternalProcess) Unsubscribe() (err error) {
	p.log.Debug("process", "Receiver finished listening to process")
	p.Terminate()

	return err
}

// PassInput delivers particular string to the process, involves locking input channel
func (p *ExternalProcess) PassInput(s string) error {
	if p.cmd.ProcessState != nil {
		return ErrProcessFinished
	}
	p.inmux.Lock()
	defer p.inmux.Unlock()
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
	bsize, backlog := int64(0), make(chan string, StdoutBufSize/100)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		for s := range backlog {
			select {
			case c <- s:
				atomic.AddInt64(&bsize, int64(-len(s)))
			case <-time.After(RcvrTimeout):
				p.Terminate()
				break
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
			atomic.AddInt64(&bsize, int64(len(str)))
		}
		if err != nil {
			p.log.Debug("process", "STDOUT stream ended: %s", err)
			break
		}
	}
	close(backlog)
	r.Close()
	wg.Wait()

	if p.terminating == 0 {
		p.wait()
	}
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
	r.Close()
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
