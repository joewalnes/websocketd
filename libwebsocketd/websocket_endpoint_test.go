package libwebsocketd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestSendWithUnknownMessageType(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
	}))
	defer s.Close()
	u := "ws" + strings.TrimPrefix(s.URL, "http")
	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer ws.Close()
	log := new(LogScope)
	log.LogFunc = func(*LogScope, LogLevel, string, string, string, ...interface{}) {}

	endpoint := NewWebSocketEndpoint(ws, false, log)
	endpoint.mtype = 100 // unknown type
	sent := endpoint.Send([]byte{})
	if sent {
		t.Fatalf("Message should not be sent")
	}
}
