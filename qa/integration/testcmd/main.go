// testcmd is a multi-tool binary used as the backend process for integration tests.
// It provides various modes (echo, count, env dump, etc.) that are cross-platform
// and don't depend on shell availability.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: testcmd <command> [args...]")
		os.Exit(1)
	}

	args := os.Args[2:]

	switch os.Args[1] {
	case "echo":
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

	case "count":
		n := intArg(args, 0, 5)
		delay := intArg(args, 1, 100)
		for i := 1; i <= n; i++ {
			fmt.Println(i)
			if i < n && delay > 0 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
		}

	case "env":
		envs := os.Environ()
		sort.Strings(envs)
		for _, e := range envs {
			fmt.Println(e)
		}

	case "env-prefix":
		prefix := strArg(args, 0, "")
		envs := os.Environ()
		sort.Strings(envs)
		for _, e := range envs {
			if strings.HasPrefix(e, prefix) {
				fmt.Println(e)
			}
		}

	case "exit":
		code := intArg(args, 0, 1)
		msg := strArg(args, 1, "")
		if msg != "" {
			fmt.Println(msg)
		}
		os.Exit(code)

	case "stderr":
		fmt.Fprintln(os.Stderr, "stderr line")
		fmt.Println("stdout line")

	case "output":
		for _, arg := range args {
			fmt.Println(arg)
		}

	case "welcome":
		msg := strArg(args, 0, "welcome")
		fmt.Println(msg)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

	case "slow-start":
		delay := intArg(args, 0, 1000)
		time.Sleep(time.Duration(delay) * time.Millisecond)
		fmt.Println("ready")
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

	case "infinite":
		delay := intArg(args, 0, 100)
		for {
			fmt.Println("tick")
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}

	case "ignore-stdin":
		fmt.Println("started")
		select {}

	case "crlf":
		fmt.Print("line1\r\n")
		fmt.Print("line2\r\n")
		fmt.Print("line3\r\n")

	case "pid-echo":
		fmt.Println(os.Getpid())
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

	case "binary-echo":
		io.Copy(os.Stdout, os.Stdin)

	case "cgi":
		body := strArg(args, 0, "Hello from CGI")
		fmt.Printf("Content-Type: text/plain\r\n")
		fmt.Printf("\r\n")
		fmt.Print(body)

	case "multi-line":
		for _, arg := range args {
			fmt.Println(arg)
		}
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func intArg(args []string, idx, defaultVal int) int {
	if idx >= len(args) {
		return defaultVal
	}
	v, err := strconv.Atoi(args[idx])
	if err != nil {
		return defaultVal
	}
	return v
}

func strArg(args []string, idx int, defaultVal string) string {
	if idx >= len(args) {
		return defaultVal
	}
	return args[idx]
}
