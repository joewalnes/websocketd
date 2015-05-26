package libwebsocketd

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"golang.org/x/net/websocket"
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

		wsconf := &websocket.Config{Version: websocket.ProtocolVersionHybi13}
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

		err = checkOrigin(wsconf, req, config, log)
		if testcase.getsErr == ReturnsError && err == nil {
			t.Errorf("Test case %#v did not get an error", testcase.name)
		} else if testcase.getsErr == ReturnsPass && err != nil {
			t.Errorf("Test case %#v got error while should've", testcase.name)
		}
	}
}

var mimetest = [][3]string{
	{"Content-type: text/plain", "Content-type", "text/plain"},
	{"Content-type:    ", "Content-type", ""},
}

func TestSplitMimeHeader(t *testing.T) {
	for _, tst := range mimetest {
		s, v := splitMimeHeader(tst[0])
		if tst[1] != s || tst[2] != v {
			t.Errorf("%v and %v  are not same as expexted %v and %v", s, v, tst[1], tst[2])
		}
	}
}
