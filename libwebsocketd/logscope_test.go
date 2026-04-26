package libwebsocketd

import (
	"testing"
)

func TestLevelFromString(t *testing.T) {
	tests := []struct {
		input string
		want  LogLevel
	}{
		{"debug", LogDebug},
		{"trace", LogTrace},
		{"access", LogAccess},
		{"info", LogInfo},
		{"error", LogError},
		{"fatal", LogFatal},
		{"none", LogNone},
		{"invalid", LogUnknown},
		{"", LogUnknown},
		{"DEBUG", LogUnknown}, // case-sensitive
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := LevelFromString(tt.input)
			if got != tt.want {
				t.Errorf("LevelFromString(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestRootLogScope(t *testing.T) {
	called := false
	logFunc := func(l *LogScope, level LogLevel, levelName, category, msg string, args ...interface{}) {
		called = true
	}

	scope := RootLogScope(LogInfo, logFunc)
	if scope.Parent != nil {
		t.Error("root scope should have nil parent")
	}
	if scope.MinLevel != LogInfo {
		t.Errorf("MinLevel = %d, want %d", scope.MinLevel, LogInfo)
	}
	if scope.Mutex == nil {
		t.Error("Mutex should not be nil")
	}
	if len(scope.Associated) != 0 {
		t.Error("Associated should be empty initially")
	}

	scope.Info("test", "hello")
	if !called {
		t.Error("LogFunc was not called")
	}
}

func TestNewLevel(t *testing.T) {
	logFunc := func(l *LogScope, level LogLevel, levelName, category, msg string, args ...interface{}) {}
	parent := RootLogScope(LogDebug, logFunc)
	parent.Associate("key", "value")

	child := parent.NewLevel(logFunc)
	if child.Parent != parent {
		t.Error("child.Parent should be parent")
	}
	if child.MinLevel != LogDebug {
		t.Errorf("child should inherit MinLevel, got %d", child.MinLevel)
	}
	if child.Mutex != parent.Mutex {
		t.Error("child should share parent's Mutex")
	}
	if len(child.Associated) != 0 {
		t.Error("child should have empty Associated (not inherited)")
	}
}

func TestAssociate(t *testing.T) {
	logFunc := func(l *LogScope, level LogLevel, levelName, category, msg string, args ...interface{}) {}
	scope := RootLogScope(LogInfo, logFunc)

	scope.Associate("url", "http://example.com")
	scope.Associate("remote", "127.0.0.1")

	if len(scope.Associated) != 2 {
		t.Fatalf("expected 2 associations, got %d", len(scope.Associated))
	}
	if scope.Associated[0].Key != "url" || scope.Associated[0].Value != "http://example.com" {
		t.Errorf("first association: %+v", scope.Associated[0])
	}
	if scope.Associated[1].Key != "remote" || scope.Associated[1].Value != "127.0.0.1" {
		t.Errorf("second association: %+v", scope.Associated[1])
	}
}

func TestLogLevelFiltering(t *testing.T) {
	var logged []string
	logFunc := func(l *LogScope, level LogLevel, levelName, category, msg string, args ...interface{}) {
		if level >= l.MinLevel {
			logged = append(logged, levelName)
		}
	}

	scope := RootLogScope(LogInfo, logFunc)

	scope.Debug("test", "should not appear")
	scope.Trace("test", "should not appear")
	scope.Access("test", "should not appear")
	scope.Info("test", "should appear")
	scope.Error("test", "should appear")
	scope.Fatal("test", "should appear")

	if len(logged) != 3 {
		t.Errorf("expected 3 logged messages (Info+Error+Fatal), got %d: %v", len(logged), logged)
	}
}

func TestTimestamp(t *testing.T) {
	ts := Timestamp()
	if ts == "" {
		t.Error("Timestamp() returned empty string")
	}
	// Should be RFC1123Z format, contains timezone offset like +0000
	if len(ts) < 20 {
		t.Errorf("Timestamp looks too short: %q", ts)
	}
}
