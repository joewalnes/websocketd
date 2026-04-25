# websocketd QA Test Plans

Comprehensive quality assurance plans for websocketd covering all features, edge cases, platform-specific behavior, and regression scenarios.

## How to Use These Plans

Each plan file covers a category of functionality. Test cases are numbered within each file using the format `CATEGORY-NNN` (e.g., `CORE-001`). Each test case includes:

- **ID**: Unique identifier
- **Title**: Brief description
- **Priority**: P0 (critical), P1 (high), P2 (medium), P3 (low)
- **Preconditions**: What must be set up before the test
- **Steps**: Exact steps to perform
- **Expected Result**: What should happen
- **Notes**: Additional context, known issues, or regression history

## Categories

| File | Category | Description |
|------|----------|-------------|
| [01-core-websocket.md](01-core-websocket.md) | Core WebSocket | Basic connection lifecycle, text/binary messaging |
| [02-process-management.md](02-process-management.md) | Process Management | Child process launching, stdio piping, termination |
| [03-cli-configuration.md](03-cli-configuration.md) | CLI & Configuration | Command-line flags, argument validation, defaults |
| [04-http-routing.md](04-http-routing.md) | HTTP & Routing | Static files, CGI, dev console, script directory mode |
| [05-security.md](05-security.md) | Security | Origin validation, TLS/SSL, environment isolation |
| [06-environment-cgi.md](06-environment-cgi.md) | Environment Variables & CGI | RFC 3875 compliance, HTTP header passthrough |
| [07-platform-windows.md](07-platform-windows.md) | Windows Platform | Windows-specific behavior, line endings, script types |
| [08-platform-unix.md](08-platform-unix.md) | Unix Platforms | macOS, Linux, FreeBSD, Solaris, ARM, containers |
| [09-protocol-compatibility.md](09-protocol-compatibility.md) | Protocol Compatibility | WebSocket versions, HTTP versions, browser testing |
| [10-edge-cases-errors.md](10-edge-cases-errors.md) | Edge Cases & Errors | Malformed input, crashes, unexpected states |
| [11-performance-scalability.md](11-performance-scalability.md) | Performance & Scalability | Concurrent connections, large messages, resource limits |
| [12-dev-console.md](12-dev-console.md) | Developer Console | Interactive testing UI functionality |
| [13-examples-languages.md](13-examples-languages.md) | Language Examples | Testing example scripts across languages |
| [14-build-release.md](14-build-release.md) | Build & Release | Cross-compilation, packaging, versioning |

## Test Environment Requirements

- **Operating Systems**: Windows 10/11, macOS (Intel + ARM), Ubuntu/Debian, FreeBSD, Solaris
- **Browsers**: Chrome (latest), Firefox (latest), Safari (latest), Edge (latest), mobile Safari, mobile Chrome
- **Languages**: Bash, Python 3, Node.js, Ruby, Perl, PHP, Go, Java, C#, Rust, Lua
- **Tools**: `wscat` or similar WebSocket CLI client, `curl`, `openssl`
- **Network**: localhost, LAN, IPv4, IPv6

## Running the Automated Tests

The existing Go test suite covers unit-level behavior:

```bash
go test ./...
```

The plans in this directory cover manual, integration, and exploratory testing that goes beyond the unit tests.
