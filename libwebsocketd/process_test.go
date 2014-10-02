package libwebsocketd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

func launchHelper(t *testing.T, args ...string) (*ExternalProcess, <-chan string) {
	cmd := helperCommand(args...)

	ep, ch, err := LaunchProcess(cmd, logger_helper(t.Log))
	if err != nil {
		t.Fatal(err.Error())
		return nil, nil
	}
	return ep, ch
}

func TestEarlyTerminate(t *testing.T) {
	ep, _ := launchHelper(t, "cat")
	ep.Terminate()
}

func chanEq(c <-chan string, data ...string) error {
	for _, m := range data {
		s, ok := <-c
		if !ok || s != m {
			return errors.New(s)
		}
	}
	return nil
}

func TestSimpleEcho(t *testing.T) {
	ep, c := launchHelper(t, "echo", "foo bar", "baz")

	if s := chanEq(c, "foo bar baz"); s != nil {
		t.Errorf("Invalid echo result %#v", s)
	}

	s, ok := <-c
	if ok || s != "" {
		t.Error("Echo returned more than one line")
	}

	time.Sleep(10 * time.Millisecond)
	if ep.cmd.ProcessState == nil {
		t.Error("Echo did not stop after sending the line")
	}
}

func TestSimpleCat(t *testing.T) {
	ep, c := launchHelper(t, "cat")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		var check string
		for i := 0; i < 3; i++ {
			s, ok := <-c
			if ok {
				check += s + "\n"
			}
		}
		if check != "foo bar\nfoo baz\nfoo bam\n" {
			t.Errorf("Invalid cat result %#v", check)
		}
	}()

	ep.PassInput("foo bar\nfoo baz\nfoo bam")

	wg.Wait()

	ep.Terminate() // this forces termination... Other way to finish is calling ep.Unsuscribe(c)

	time.Sleep(10 * time.Millisecond)
	if ep.cmd.ProcessState == nil {
		t.Error("Cat did not stop after termination")
	}
}

func TestSlowCat(t *testing.T) {
	ep, c := launchHelper(t, "cat")

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		var check string
		for i := 0; i < 3; i++ {
			s, ok := <-c
			if ok {
				check += s + "\n"
			}
		}
		if check != "foo bar\nfoo baz\nfoo bam\n" {
			t.Errorf("Invalid cat result %#v", check)
		}
	}()

	ep.PassInput("foo bar\nfoo baz\nfoo bam")

	wg.Wait()

	ep.Terminate() // this forces termination... Other way to finish is calling ep.Unsuscribe(c)

	time.Sleep(10 * time.Millisecond)
	if ep.cmd.ProcessState == nil {
		t.Error("Cat did not stop after termination")
	}
}

// ---
//
// following is copied from standard lib see  http://golang.org/src/pkg/os/exec/exec_test.go
// (c) 2009 The Go Authors. All rights reserved. For more information see http://golang.org/LICENSE
//
func helperCommand(s ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, s...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

const stdinCloseTestString = "Some test string."

// TestHelperProcess isn't a real test. It's used as a helper process
// for TestParameterRun.
func TestHelperProcess(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	// Determine which command to use to display open files.
	ofcmd := "lsof"
	switch runtime.GOOS {
	case "dragonfly", "freebsd", "netbsd", "openbsd":
		ofcmd = "fstat"
	}

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "echo":
		iargs := []interface{}{}
		for _, s := range args {
			iargs = append(iargs, s)
		}
		fmt.Println(iargs...)
	case "cat":
		if len(args) == 0 {
			io.Copy(os.Stdout, os.Stdin)
			return
		}
		exit := 0
		for _, fn := range args {
			f, err := os.Open(fn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				exit = 2
			} else {
				defer f.Close()
				io.Copy(os.Stdout, f)
			}
		}
		os.Exit(exit)
	case "pipetest":
		bufr := bufio.NewReader(os.Stdin)
		for {
			line, _, err := bufr.ReadLine()
			if err == io.EOF {
				break
			} else if err != nil {
				os.Exit(1)
			}
			if bytes.HasPrefix(line, []byte("O:")) {
				os.Stdout.Write(line)
				os.Stdout.Write([]byte{'\n'})
			} else if bytes.HasPrefix(line, []byte("E:")) {
				os.Stderr.Write(line)
				os.Stderr.Write([]byte{'\n'})
			} else {
				os.Exit(1)
			}
		}
	case "stdinClose":
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if s := string(b); s != stdinCloseTestString {
			fmt.Fprintf(os.Stderr, "Error: Read %q, want %q", s, stdinCloseTestString)
			os.Exit(1)
		}
		os.Exit(0)
	case "read3": // read fd 3
		fd3 := os.NewFile(3, "fd3")
		bs, err := ioutil.ReadAll(fd3)
		if err != nil {
			fmt.Printf("ReadAll from fd 3: %v", err)
			os.Exit(1)
		}
		switch runtime.GOOS {
		case "dragonfly":
			// TODO(jsing): Determine why DragonFly is leaking
			// file descriptors...
		case "darwin":
			// TODO(bradfitz): broken? Sometimes.
			// http://golang.org/issue/2603
			// Skip this additional part of the test for now.
		case "netbsd":
			// TODO(jsing): This currently fails on NetBSD due to
			// the cloned file descriptors that result from opening
			// /dev/urandom.
			// http://golang.org/issue/3955
		default:
			// Now verify that there are no other open fds.
			var files []*os.File
			for wantfd := basefds() + 1; wantfd <= 100; wantfd++ {
				f, err := os.Open(os.Args[0])
				if err != nil {
					fmt.Printf("error opening file with expected fd %d: %v", wantfd, err)
					os.Exit(1)
				}
				if got := f.Fd(); got != wantfd {
					fmt.Printf("leaked parent file. fd = %d; want %d\n", got, wantfd)
					out, _ := exec.Command(ofcmd, "-p", fmt.Sprint(os.Getpid())).CombinedOutput()
					fmt.Print(string(out))
					os.Exit(1)
				}
				files = append(files, f)
			}
			for _, f := range files {
				f.Close()
			}
		}
		// Referring to fd3 here ensures that it is not
		// garbage collected, and therefore closed, while
		// executing the wantfd loop above.  It doesn't matter
		// what we do with fd3 as long as we refer to it;
		// closing it is the easy choice.
		fd3.Close()
		os.Stdout.Write(bs)
	case "exit":
		n, _ := strconv.Atoi(args[0])
		os.Exit(n)
	case "describefiles":
		f := os.NewFile(3, fmt.Sprintf("fd3"))
		ln, err := net.FileListener(f)
		if err == nil {
			fmt.Printf("fd3: listener %s\n", ln.Addr())
			ln.Close()
		}
		os.Exit(0)
	case "extraFilesAndPipes":
		n, _ := strconv.Atoi(args[0])
		pipes := make([]*os.File, n)
		for i := 0; i < n; i++ {
			pipes[i] = os.NewFile(uintptr(3+i), strconv.Itoa(i))
		}
		response := ""
		for i, r := range pipes {
			ch := make(chan string, 1)
			go func(c chan string) {
				buf := make([]byte, 10)
				n, err := r.Read(buf)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Child: read error: %v on pipe %d\n", err, i)
					os.Exit(1)
				}
				c <- string(buf[:n])
				close(c)
			}(ch)
			select {
			case m := <-ch:
				response = response + m
			case <-time.After(5 * time.Second):
				fmt.Fprintf(os.Stderr, "Child: Timeout reading from pipe: %d\n", i)
				os.Exit(1)
			}
		}
		fmt.Fprintf(os.Stderr, "child: %s", response)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}

// basefds returns the number of expected file descriptors
// to be present in a process at start.
func basefds() uintptr {
	n := os.Stderr.Fd() + 1

	// Go runtime for 32-bit Plan 9 requires that /dev/bintime
	// be kept open.
	// See ../../runtime/time_plan9_386.c:/^runtime·nanotime
	if runtime.GOOS == "plan9" && runtime.GOARCH == "386" {
		n++
	}
	return n
}
