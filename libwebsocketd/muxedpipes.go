// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

var MuxedPipes = map[string]*MuxedPipe{}
var muxlock = new(sync.Mutex)

type MuxedPipe struct {
	cmd     string
	log     *LogScope
	bin     bool
	wes     map[chan []byte]bool
	process Endpoint
	sync.RWMutex
}

func MuxedLaunchCmd(wsh *WebsocketdHandler, log *LogScope) *MuxedPipe {
	muxlock.Lock()
	defer muxlock.Unlock()

	muxed, ok := MuxedPipes[wsh.command]
	if ok {
		return muxed
	}
	muxed = &MuxedPipe{
		cmd: wsh.command,
		log: log,
		wes: make(map[chan []byte]bool),
	}
	MuxedPipes[wsh.command] = muxed

	launched, err := launchCmd(wsh.command, wsh.server.Config.CommandArgs, wsh.Env)
	if err != nil {
		log.Error("muxed process", "Could not launch process %s %s (%s)",
			wsh.command, strings.Join(wsh.server.Config.CommandArgs, " "),
			err,
		)
		return nil
	}

	log.Associate("pid", strconv.Itoa(launched.cmd.Process.Pid))
	bin := wsh.server.Config.Binary
	process := NewProcessEndpoint(launched, bin, log)
	if cms := wsh.server.Config.CloseMs; cms != 0 {
		process.closetime += time.Duration(cms) * time.Millisecond
	}

	muxed.process = process
	muxed.bin = bin

	go PipeMuxedEndpoints(muxed)
	return muxed
}

func PipeMuxedEndpoints(muxed *MuxedPipe) {
	p := muxed.process
	p.StartReading()

	defer func() {
		muxlock.Lock()
		m := MuxedPipes[muxed.cmd]
		if m == muxed {
			delete(MuxedPipes, muxed.cmd)
		}
		muxed.RLock()
		for c := range muxed.wes {
			close(c)
		}
		muxed.RUnlock()

		p.Terminate()
		muxlock.Unlock()
	}()

	for {
		msg, ok := <-p.Output()
		if !ok {
			return
		}

		muxed.RLock()
		for c := range muxed.wes {
			c <- msg
		}
		if len(muxed.wes) == 0 {
			muxlock.Lock()
			delete(MuxedPipes, muxed.cmd)
			muxlock.Unlock()
			return
		}
		muxed.RUnlock()
	}
}

func MuxedAttach(muxed *MuxedPipe, key string, e Endpoint) {
	c := make(chan []byte)
	muxed.Lock()
	muxed.wes[c] = true
	muxed.Unlock()
	defer func() {
		muxed.Lock()
		delete(muxed.wes, c)
		muxed.Unlock()
	}()

	for {
		msg, ok := <-c
		if !ok || !e.Send(msg) {
			return
		}
	}
}
