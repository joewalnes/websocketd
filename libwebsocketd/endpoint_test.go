package libwebsocketd

import (
	"strconv"
	"sync"
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
		answ := trimEOL([]byte(eol_tests[n]))
		if string(answ) != eol_answers[n] {
			t.Errorf("Answer '%s' did not match predicted '%s'", answ, eol_answers[n])
		}
	}
}

func BenchmarkTrimEOL(b *testing.B) {
	for n := 0; n < b.N; n++ {
		trimEOL([]byte(eol_tests[n%len(eol_tests)]))
	}
}

type TestEndpoint struct {
	limit  int
	prefix string
	c      chan []byte
	mu     sync.Mutex
	result []string
}

func (e *TestEndpoint) StartReading() {
	go func() {
		for i := 0; i < e.limit; i++ {
			e.c <- []byte(e.prefix + strconv.Itoa(i))
		}
		close(e.c)
	}()
}

func (e *TestEndpoint) Terminate() {
}

func (e *TestEndpoint) Output() chan []byte {
	return e.c
}

func (e *TestEndpoint) Send(msg []byte) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.result = append(e.result, string(msg))
	return true
}

func (e *TestEndpoint) Results() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := make([]string, len(e.result))
	copy(cp, e.result)
	return cp
}

func TestEndpointPipe(t *testing.T) {
	one := &TestEndpoint{2, "one:", make(chan []byte), sync.Mutex{}, make([]string, 0)}
	two := &TestEndpoint{4, "two:", make(chan []byte), sync.Mutex{}, make([]string, 0)}
	PipeEndpoints(one, two)

	oneResults := one.Results()
	twoResults := two.Results()

	if len(oneResults) != 4 || len(twoResults) != 2 {
		t.Errorf("Invalid lengths, should be 4 and 2: %v %v", oneResults, twoResults)
	} else if oneResults[0] != "two:0" || twoResults[0] != "one:0" {
		t.Errorf("Invalid first results, should be two:0 and one:0: %#v %#v", oneResults[0], twoResults[0])
	}
}

func TestEndpointPipeBidirectional(t *testing.T) {
	// Both endpoints send and receive concurrently
	one := &TestEndpoint{10, "one:", make(chan []byte), sync.Mutex{}, make([]string, 0)}
	two := &TestEndpoint{10, "two:", make(chan []byte), sync.Mutex{}, make([]string, 0)}
	PipeEndpoints(one, two)

	oneResults := one.Results()
	twoResults := two.Results()

	if len(oneResults) != 10 {
		t.Errorf("endpoint one should have received 10 messages, got %d", len(oneResults))
	}
	if len(twoResults) != 10 {
		t.Errorf("endpoint two should have received 10 messages, got %d", len(twoResults))
	}
}

func TestEndpointPipeOneDirectionCloses(t *testing.T) {
	// One endpoint sends nothing (closes immediately), other sends messages
	silent := &TestEndpoint{0, "", make(chan []byte), sync.Mutex{}, make([]string, 0)}
	talker := &TestEndpoint{5, "talk:", make(chan []byte), sync.Mutex{}, make([]string, 0)}

	done := make(chan struct{})
	go func() {
		PipeEndpoints(silent, talker)
		close(done)
	}()

	select {
	case <-done:
		// Good — PipeEndpoints returned after silent closed
	case <-time.After(5 * time.Second):
		t.Fatal("PipeEndpoints did not return after one endpoint closed")
	}
}

type FailingSendEndpoint struct {
	TestEndpoint
	failAfter int
	sendCount int
}

func (e *FailingSendEndpoint) Send(msg []byte) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sendCount++
	if e.sendCount > e.failAfter {
		return false
	}
	e.result = append(e.result, string(msg))
	return true
}

func TestEndpointPipeSendFailure(t *testing.T) {
	// Receiver rejects after 2 messages — PipeEndpoints should exit
	sender := &TestEndpoint{10, "msg:", make(chan []byte), sync.Mutex{}, make([]string, 0)}
	receiver := &FailingSendEndpoint{
		TestEndpoint: TestEndpoint{0, "", make(chan []byte), sync.Mutex{}, make([]string, 0)},
		failAfter:    2,
	}

	done := make(chan struct{})
	go func() {
		PipeEndpoints(sender, receiver)
		close(done)
	}()

	select {
	case <-done:
		results := receiver.Results()
		if len(results) > 2 {
			t.Errorf("receiver should have at most 2 messages, got %d", len(results))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("PipeEndpoints did not return after Send failure")
	}
}
