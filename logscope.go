// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"sync"
	"time"
)

type LogLevel int

const (
	logDebug = iota
	logTrace
	logAccess
	logInfo
	logError
	logFatal
)

type LogScope struct {
	parent     *LogScope   // Parent scope
	minLevel   LogLevel    // Minimum log level to write out.
	mutex      *sync.Mutex // Should be shared across all LogScopes that write to the same destination.
	associated []AssocPair // Additional data associated with scope
}

type AssocPair struct {
	key   string
	value string
}

func (l *LogScope) Associate(key string, value string) {
	l.associated = append(l.associated, AssocPair{key, value})
}

func (l *LogScope) Debug(category string, msg string, args ...interface{}) {
	l.log(logDebug, "DEBUG", category, msg, args...)
}

func (l *LogScope) Trace(category string, msg string, args ...interface{}) {
	l.log(logTrace, "TRACE", category, msg, args...)
}

func (l *LogScope) Access(category string, msg string, args ...interface{}) {
	l.log(logAccess, "ACCESS", category, msg, args...)
}

func (l *LogScope) Info(category string, msg string, args ...interface{}) {
	l.log(logInfo, "INFO", category, msg, args...)
}

func (l *LogScope) Error(category string, msg string, args ...interface{}) {
	l.log(logError, "ERROR", category, msg, args...)
}

func (l *LogScope) Fatal(category string, msg string, args ...interface{}) {
	l.log(logFatal, "FATAL", category, msg, args...)
}

func (l *LogScope) log(level LogLevel, levelName string, category string, msg string, args ...interface{}) {
	if level < l.minLevel {
		return
	}
	fullMsg := fmt.Sprintf(msg, args...)

	assocDump := ""
	for index, pair := range l.associated {
		if index > 0 {
			assocDump += " "
		}
		assocDump += fmt.Sprintf("%s:'%s'", pair.key, pair.value)
	}

	l.mutex.Lock()
	fmt.Printf("%s | %-6s | %-10s | %s | %s\n", Timestamp(), levelName, category, assocDump, fullMsg)
	l.mutex.Unlock()
}

func (parent *LogScope) NewLevel() *LogScope {
	return &LogScope{
		parent:     parent,
		minLevel:   parent.minLevel,
		mutex:      parent.mutex,
		associated: make([]AssocPair, 0)}
}

func RootLogScope(minLevel LogLevel) *LogScope {
	return &LogScope{
		parent:     nil,
		minLevel:   minLevel,
		mutex:      &sync.Mutex{},
		associated: make([]AssocPair, 0)}
}

func Timestamp() string {
	return time.Now().Format(time.RFC1123Z)
}
