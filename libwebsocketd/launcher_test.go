package libwebsocketd

import (
	"runtime"
	"testing"
)

func TestLaunchCmd(t *testing.T) {
	t.Run("valid command", func(t *testing.T) {
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "cmd.exe"
		} else {
			cmd = "/bin/echo"
		}
		lp, err := launchCmd(cmd, []string{"hello"}, []string{})
		if err != nil {
			t.Fatalf("launchCmd failed: %v", err)
		}
		if lp.cmd == nil {
			t.Fatal("cmd is nil")
		}
		if lp.stdin == nil {
			t.Fatal("stdin is nil")
		}
		if lp.stdout == nil {
			t.Fatal("stdout is nil")
		}
		if lp.stderr == nil {
			t.Fatal("stderr is nil")
		}
		lp.stdin.Close()
		lp.cmd.Wait()
	})

	t.Run("nonexistent command", func(t *testing.T) {
		_, err := launchCmd("/nonexistent/command/xyz", nil, nil)
		if err == nil {
			t.Fatal("expected error for nonexistent command")
		}
	})

	t.Run("environment passed to process", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("test uses /bin/sh")
		}
		env := []string{"TEST_WSD_LAUNCH=hello"}
		lp, err := launchCmd("/bin/sh", []string{"-c", "echo $TEST_WSD_LAUNCH"}, env)
		if err != nil {
			t.Fatalf("launchCmd failed: %v", err)
		}
		buf := make([]byte, 100)
		n, _ := lp.stdout.Read(buf)
		output := string(buf[:n])
		if output != "hello\n" {
			t.Errorf("expected 'hello\\n', got %q", output)
		}
		lp.stdin.Close()
		lp.cmd.Wait()
	})

	t.Run("pipes are functional", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("test uses cat")
		}
		lp, err := launchCmd("/bin/cat", nil, []string{})
		if err != nil {
			t.Fatalf("launchCmd failed: %v", err)
		}

		// Write to stdin
		_, err = lp.stdin.Write([]byte("test input\n"))
		if err != nil {
			t.Fatalf("stdin write failed: %v", err)
		}

		// Read from stdout
		buf := make([]byte, 100)
		n, err := lp.stdout.Read(buf)
		if err != nil {
			t.Fatalf("stdout read failed: %v", err)
		}
		if string(buf[:n]) != "test input\n" {
			t.Errorf("expected 'test input\\n', got %q", string(buf[:n]))
		}

		lp.stdin.Close()
		lp.cmd.Wait()
	})
}
