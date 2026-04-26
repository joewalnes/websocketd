package libwebsocketd

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

var tellHostPortTests = []struct {
	src          string
	ssl          bool
	server, port string
}{
	{"localhost", false, "localhost", "80"},
	{"localhost:8080", false, "localhost", "8080"},
	{"localhost", true, "localhost", "443"},
	{"localhost:8080", true, "localhost", "8080"},
}

func TestTellHostPort(t *testing.T) {
	for _, testcase := range tellHostPortTests {
		s, p, e := tellHostPort(testcase.src, testcase.ssl)
		if testcase.server == "" {
			if e == nil {
				t.Errorf("test case for %#v failed, error was not returned", testcase.src)
			}
		} else if e != nil {
			t.Errorf("test case for %#v failed, error should not happen", testcase.src)
		}
		if testcase.server != s || testcase.port != p {
			t.Errorf("test case for %#v failed, server or port mismatch to expected values (%s:%s)", testcase.src, s, p)
		}
	}
}

var NoOriginsAllowed = []string{}
var NoOriginList []string = nil

const (
	ReqHTTPS = iota
	ReqHTTP
	OriginMustBeSame
	OriginCouldDiffer
	ReturnsPass
	ReturnsError
)

var CheckOriginTests = []struct {
	host    string
	reqtls  int
	origin  string
	same    int
	allowed []string
	getsErr int
	name    string
}{
	{"server.example.com", ReqHTTP, "http://example.com", OriginCouldDiffer, NoOriginList, ReturnsPass, "any origin allowed"},
	{"server.example.com", ReqHTTP, "http://example.com", OriginMustBeSame, NoOriginList, ReturnsError, "same origin mismatch"},
	{"server.example.com", ReqHTTP, "http://server.example.com", OriginMustBeSame, NoOriginList, ReturnsPass, "same origin match"},
	{"server.example.com", ReqHTTP, "https://server.example.com", OriginMustBeSame, NoOriginList, ReturnsError, "same origin schema mismatch 1"},
	{"server.example.com", ReqHTTPS, "http://server.example.com", OriginMustBeSame, NoOriginList, ReturnsError, "same origin schema mismatch 2"},
	{"server.example.com", ReqHTTP, "http://example.com", OriginCouldDiffer, NoOriginsAllowed, ReturnsError, "no origins allowed"},
	{"server.example.com", ReqHTTP, "http://example.com", OriginCouldDiffer, []string{"server.example.com"}, ReturnsError, "no origin allowed matches (junk prefix)"},
	{"server.example.com", ReqHTTP, "http://example.com", OriginCouldDiffer, []string{"example.com.t"}, ReturnsError, "no origin allowed matches (junk suffix)"},
	{"server.example.com", ReqHTTP, "http://example.com", OriginCouldDiffer, []string{"example.com"}, ReturnsPass, "origin allowed clean match"},
	{"server.example.com", ReqHTTP, "http://example.com:81", OriginCouldDiffer, []string{"example.com"}, ReturnsPass, "origin allowed any port match"},
	{"server.example.com", ReqHTTP, "http://example.com", OriginCouldDiffer, []string{"example.com:80"}, ReturnsPass, "origin allowed port match"},
	{"server.example.com", ReqHTTP, "http://example.com", OriginCouldDiffer, []string{"example.com:81"}, ReturnsError, "origin allowed port mismatch"},
	{"server.example.com", ReqHTTP, "http://example.com", OriginCouldDiffer, []string{"example.com:81"}, ReturnsError, "origin allowed port mismatch"},
	{"server.example.com", ReqHTTP, "http://example.com:81", OriginCouldDiffer, []string{"example.com:81"}, ReturnsPass, "origin allowed port 81 match"},
	{"server.example.com", ReqHTTP, "null", OriginCouldDiffer, NoOriginList, ReturnsPass, "any origin allowed, even null"},
	{"server.example.com", ReqHTTP, "", OriginCouldDiffer, NoOriginList, ReturnsPass, "any origin allowed, even empty"},
}

func TestCheckOrigin(t *testing.T) {
	for _, testcase := range CheckOriginTests {
		br := bufio.NewReader(strings.NewReader(fmt.Sprintf(`GET /chat HTTP/1.1
Host: %s
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Origin: %s
Sec-WebSocket-Version: 13

`, testcase.host, testcase.origin)))

		req, err := http.ReadRequest(br)
		if err != nil {
			t.Fatal("request", err)
		}

		log := new(LogScope)
		log.LogFunc = func(*LogScope, LogLevel, string, string, string, ...interface{}) {}

		config := new(Config)

		if testcase.reqtls == ReqHTTPS { // Fake TLS
			config.Ssl = true
			req.TLS = &tls.ConnectionState{}
		}
		if testcase.same == OriginMustBeSame {
			config.SameOrigin = true
		}
		if testcase.allowed != nil {
			config.AllowOrigins = testcase.allowed
		}

		err = checkOrigin(req, config, log)
		if testcase.getsErr == ReturnsError && err == nil {
			t.Errorf("Test case %#v did not get an error", testcase.name)
		} else if testcase.getsErr == ReturnsPass && err != nil {
			t.Errorf("Test case %#v got error while expected to pass", testcase.name)
		}
	}
}

var mimetest = [][3]string{
	{"Content-Type: text/plain", "Content-Type", "text/plain"},
	{"Content-Type:    ", "Content-Type", ""},
}

func TestSplitMimeHeader(t *testing.T) {
	for _, tst := range mimetest {
		s, v := splitMimeHeader(tst[0])
		if tst[1] != s || tst[2] != v {
			t.Errorf("%v and %v  are not same as expexted %v and %v", s, v, tst[1], tst[2])
		}
	}
}

// --- New unit tests for extracted functions ---

func TestIsWebSocketUpgrade(t *testing.T) {
	tests := []struct {
		name       string
		upgrade    string
		connection string
		want       bool
	}{
		{"standard", "websocket", "Upgrade", true},
		{"lowercase", "websocket", "upgrade", true},
		{"mixed case upgrade header", "WebSocket", "Upgrade", true},
		{"connection with multiple values", "websocket", "keep-alive, Upgrade", true},
		{"no upgrade header", "", "Upgrade", false},
		{"wrong upgrade value", "h2c", "Upgrade", false},
		{"no connection header", "websocket", "", false},
		{"connection without upgrade", "websocket", "keep-alive", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.upgrade != "" {
				req.Header.Set("Upgrade", tt.upgrade)
			}
			if tt.connection != "" {
				req.Header.Set("Connection", tt.connection)
			}
			if got := isWebSocketUpgrade(req); got != tt.want {
				t.Errorf("isWebSocketUpgrade() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchOrigin(t *testing.T) {
	tests := []struct {
		name       string
		server     string
		port       string
		scheme     string
		allowed    []string
		wantMatch  bool
	}{
		{"exact host match", "example.com", "80", "http", []string{"example.com"}, true},
		{"host with any port", "example.com", "8080", "http", []string{"example.com"}, true},
		{"exact host and port", "example.com", "81", "http", []string{"example.com:81"}, true},
		{"port mismatch", "example.com", "80", "http", []string{"example.com:81"}, false},
		{"host mismatch", "other.com", "80", "http", []string{"example.com"}, false},
		{"multiple allowed", "b.com", "80", "http", []string{"a.com", "b.com"}, true},
		{"scheme match", "example.com", "443", "https", []string{"https://example.com:443"}, true},
		{"scheme mismatch", "example.com", "80", "http", []string{"https://example.com"}, false},
		{"empty list", "example.com", "80", "http", []string{}, false},
		{"short origin string", "x", "80", "http", []string{"x"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchOrigin(tt.server, tt.port, tt.scheme, tt.allowed)
			if got != tt.wantMatch {
				t.Errorf("matchOrigin(%q, %q, %q, %v) = %v, want %v",
					tt.server, tt.port, tt.scheme, tt.allowed, got, tt.wantMatch)
			}
		})
	}
}

func TestTellURL(t *testing.T) {
	tests := []struct {
		name   string
		ssl    bool
		scheme string
		host   string
		path   string
		want   string
	}{
		{"http basic", false, "http", "localhost:8080", "/path", "http://localhost:8080/path"},
		{"https basic", true, "http", "localhost:8080", "/path", "https://localhost:8080/path"},
		{"ws scheme", false, "ws", "localhost:8080", "/ws", "ws://localhost:8080/ws"},
		{"wss scheme", true, "ws", "localhost:8080", "/ws", "wss://localhost:8080/ws"},
		{"port-only host uses hostname", false, "http", ":8080", "/", "http://testhost:8080/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &WebsocketdServer{
				Config:   &Config{Ssl: tt.ssl},
				hostname: "testhost",
			}
			got := s.TellURL(tt.scheme, tt.host, tt.path)
			if got != tt.want {
				t.Errorf("TellURL(%q, %q, %q) = %q, want %q", tt.scheme, tt.host, tt.path, got, tt.want)
			}
		})
	}
}

func TestNoteForkCreatedAndCompleted(t *testing.T) {
	t.Run("nil forks (unlimited)", func(t *testing.T) {
		s := &WebsocketdServer{}
		if err := s.noteForkCreated(); err != nil {
			t.Errorf("unlimited forks should never fail: %v", err)
		}
		s.noteForkCompleted() // should not panic
	})

	t.Run("fork limit enforced", func(t *testing.T) {
		s := &WebsocketdServer{forks: make(chan byte, 2)}

		// Fill up forks
		if err := s.noteForkCreated(); err != nil {
			t.Fatal(err)
		}
		if err := s.noteForkCreated(); err != nil {
			t.Fatal(err)
		}

		// Third should fail
		if err := s.noteForkCreated(); err != ErrForkNotAllowed {
			t.Errorf("expected ErrForkNotAllowed, got %v", err)
		}

		// Release one, should work again
		s.noteForkCompleted()
		if err := s.noteForkCreated(); err != nil {
			t.Errorf("fork should be available after release: %v", err)
		}

		// Clean up
		s.noteForkCompleted()
		s.noteForkCompleted()
	})
}

func TestPushHeaders(t *testing.T) {
	h := http.Header{}
	pushHeaders(h, []string{"X-Foo: bar", "X-Baz: qux"})

	if got := h.Get("X-Foo"); got != "bar" {
		t.Errorf("X-Foo = %q, want %q", got, "bar")
	}
	if got := h.Get("X-Baz"); got != "qux" {
		t.Errorf("X-Baz = %q, want %q", got, "qux")
	}
}
