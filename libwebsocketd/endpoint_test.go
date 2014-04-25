package libwebsocketd

import (
	"strconv"
	"testing"
	"time"
)

var eol_tests = []string{
	"", "\n", "\r\n", "ok\n", "ok\n",
	"quite long string for our test\n",
	"quite long string for our test\r\n",
}

var eol_answers = []string{
	"", "", "", "ok", "ok",
	"quite long string for our test", "quite long string for our test",
}

func TestTrimEOL(t *testing.T) {
	for n := 0; n < len(eol_tests); n++ {
		answ := trimEOL(eol_tests[n])
		if answ != eol_answers[n] {
			t.Errorf("Answer '%s' did not match predicted '%s'", answ, eol_answers[n])
		}
	}
}

func BenchmarkTrimEOL(b *testing.B) {
	for n := 0; n < b.N; n++ {
		trimEOL(eol_tests[n%len(eol_tests)])
	}
}

type TestEndpoint struct {
	limit  int
	prefix string
	c      chan string
	result []string
}

func (e *TestEndpoint) StartReading() {
	go func() {
		for i := 0; i < e.limit; i++ {
			e.c <- e.prefix + strconv.Itoa(i)
		}
		time.Sleep(time.Millisecond) // should be enough for smaller channel to catch up with long one
		close(e.c)
	}()
}

func (e *TestEndpoint) Terminate() {
}

func (e *TestEndpoint) Output() chan string {
	return e.c
}

func (e *TestEndpoint) Send(msg string) bool {
	e.result = append(e.result, msg)
	return true
}

func TestEndpointPipe(t *testing.T) {
	one := &TestEndpoint{2, "one:", make(chan string), make([]string, 0)}
	two := &TestEndpoint{4, "two:", make(chan string), make([]string, 0)}
	PipeEndpoints(one, two)
	if len(one.result) != 4 || len(two.result) != 2 {
		t.Errorf("Invalid lengths, should be 4 and 2: %v %v", one.result, two.result)
	} else if one.result[0] != "two:0" || two.result[0] != "one:0" {
		t.Errorf("Invalid first results, should be two:0 and one:0: %#v %#v", one.result[0], two.result[0])
	}
}
