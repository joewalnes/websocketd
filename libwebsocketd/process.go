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
	"time"
)

var ErrNoConsumers = errors.New("All consumers are gone")
var ErrProcessFinished = errors.New("Process already finished")
var ErrUnknownConsumer = errors.New("No consumer to unsubscribe")

// RcvrTimeout is a very short duration to determine if subscriber is unable to process data quickly enough.
// Zero is not practical because it would cause packet receiving to block while OS passes data via Pipe to process.
var RcvrTimeout = time.Millisecond

// ExternalProcess holds info about running process and sends info to subscribers using channels
type ExternalProcess struct {
	cmd *exec.Cmd

	consumers []chan string
	cmux      *sync.Mutex

	in        io.Writer
	term_once func()
	inmux     *sync.Mutex

	log *LogScope
}

// LaunchProcess initializes ExternalProcess struct fields
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

	firstconsumer := make(chan string)
	p := &ExternalProcess{
		cmd:       cmd,
		consumers: []chan string{firstconsumer},
		cmux:      &sync.Mutex{},
		in:        stdin,
		inmux:     &sync.Mutex{},
		log:       log,
	}
	log.Associate("pid", strconv.Itoa(p.Pid()))
	p.log.Trace("process", "Command started, first consumer channel created")

	p.term_once = func() {
		p.term_once = func() {}

		// wait for program to end
		p.cmd.Wait()

		// close channels
		if l := len(p.consumers); l > 0 {
			p.log.Trace("process", "Closing %d consumer channels", l)
			for _, x := range p.consumers {
				close(x)
			}
		}
		p.consumers = nil

		// close pipes
		stdin.Close()
		stdout.Close()
		stderr.Close()

		p.log.Debug("process", "Process completed, status: %s", p.cmd.ProcessState.String())
	}

	// Run output listeners
	go p.process_stdout(stdout)
	go p.process_stderr(stderr)

	return p, firstconsumer, nil
}

// Terminate tries to stop process forcefully using interrupt and kill signals with a second of waiting time between them. If the kill is unsuccessful, it might be repeated
// again and again while system accepts those attempts.
func (p *ExternalProcess) Terminate() {
	if p.cmd.ProcessState == nil {
		p.log.Debug("process", "Sending SIGINT to %d", p.Pid())

		term := make(chan int)
		go func() { p.term_once(); close(term) }()

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

// Subscribe allows someone to open process's
func (p *ExternalProcess) Subscribe() (<-chan string, error) {
	p.cmux.Lock()
	defer p.cmux.Unlock()
	if p.consumers == nil {
		return nil, ErrProcessFinished
	}
	p.log.Trace("process", "New consumer added")
	c := make(chan string)
	p.consumers = append(p.consumers, c)
	return c, nil
}

// Unubscribe signals back from the consumer and helps to finish process if output is quiet and all subscribers disconnected
func (p *ExternalProcess) Unsubscribe(x <-chan string) (err error) {
	p.cmux.Lock()
	defer p.cmux.Unlock()

	if p.consumers != nil {
		// we did not terminate consumers yet
		ln := len(p.consumers)
		if ln == 1 {
			// simple choice!
			if p.consumers[0] == x {
				p.log.Debug("process", "No receivers listen, last one unsubscribed")
				p.Terminate()
			} else {
				err = ErrUnknownConsumer
			}
		} else {
			for i, m := range p.consumers {
				if m == x {
					p.log.Trace("process", "Process subscriber unsubscribed leaving %d to listen", ln-1)
					close(m)
					copy(p.consumers[i:], p.consumers[i+1:])
					p.consumers = p.consumers[:ln-1]
					break
				}
			}
			// error if nothing changed
			if len(p.consumers) == ln {
				err = ErrUnknownConsumer
			}
		}
	} else {
		err = ErrNoConsumers
	}
	return err
}

// demux_content delivers particular string to all consumers
func (p *ExternalProcess) demux_content(s string) error {
	p.cmux.Lock()
	defer p.cmux.Unlock()

	ln := len(p.consumers)
	alive := make([]bool, ln)

	// Idea here is to run parallel send to consumers with same timeout.
	// All of those sends will put their result into same pre-allocated slice.
	// This could be changed to a channel later to avoid blocking.
	wg := sync.WaitGroup{}
	for i := range p.consumers {
		wg.Add(1)
		go func(i int) {
			select {
			case p.consumers[i] <- s:
				p.log.Trace("process", "Sent process output %#v to %v", s, i)
				alive[i] = true
			case <-time.After(RcvrTimeout):
				// consumer cannot receive data, removing it (note, sometimes it's ok to have small delay)
				p.log.Debug("process", "Dropped message '%s' to consumer %d, closing it", s, i)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	d := 0 // counter for deletions
	for j := 0; j < ln; j++ {
		if !alive[j] {
			i := j - d
			close(p.consumers[i])
			copy(p.consumers[i:], p.consumers[i+1:])
			d++
			p.consumers = p.consumers[:ln-d]
		}
	}

	if d == ln { // all consumers gone
		return ErrNoConsumers
	} else {
		return nil
	}
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
func (p *ExternalProcess) process_stdout(r io.ReadCloser) {
	buf := bufio.NewReader(r)
	for {
		str, err := buf.ReadString('\n')
		str = trimEOL(str)
		if str != "" {
			snderr := p.demux_content(str)
			if snderr != nil {
				break
			}
		}
		if err != nil {
			p.log.Debug("process", "STDOUT stream ended: %s", err)
			break
		}
	}
	p.cmux.Lock()
	defer p.cmux.Unlock()

	// reuse reading thread for waiting for process to finish
	p.term_once()
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
