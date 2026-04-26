package integration

// Known bug tests — each was verified to fail before the fix was applied,
// then the fix was made, and the skip removed.
//
// TestBUG001 — fixed: --header now applies to both HTTP and WS responses.
// TestBUG002 — fixed: GATEWAY_INTERFACE set to standard "CGI/1.1".
// TestBUG003 — dropped: not a bug, standard Go http.FileServer behavior.
// TestBUG004 — fixed: gorilla/websocket v1.4.0 → v1.5.3.
// TestBUG005 — fixed: go.mod updated from Go 1.15 to Go 1.21.
// TestBUG006 — fixed: Send() made non-blocking to prevent pipe deadlock.
//
// Active regression tests for these fixes now live in their respective
// test files (cli_test.go, env_test.go, bug006_test.go, etc.)
