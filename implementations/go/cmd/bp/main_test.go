package main

import (
	"os"
	"testing"
)

// withSilencedStdout runs fn with os.Stdout redirected to /dev/null so command
// output does not clutter test logs.
func withSilencedStdout(t *testing.T, fn func()) {
	t.Helper()
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer func() { _ = devnull.Close() }()
	saved := os.Stdout
	os.Stdout = devnull
	defer func() { os.Stdout = saved }()
	fn()
}

func TestDemoCommand(t *testing.T) {
	withSilencedStdout(t, func() {
		if err := cmdDemo([]string{"--net", t.TempDir()}); err != nil {
			t.Fatalf("cmdDemo: %v", err)
		}
	})
}

func TestSendThenInbox(t *testing.T) {
	dir := t.TempDir()
	withSilencedStdout(t, func() {
		if err := cmdSend([]string{"--net", dir, "--world", "w", "--pass", "p", "--audience", "1", "--text", "hello"}); err != nil {
			t.Fatalf("cmdSend: %v", err)
		}
		if err := cmdInbox([]string{"--net", dir, "--world", "w", "--audience", "1"}); err != nil {
			t.Fatalf("cmdInbox: %v", err)
		}
	})
}

func TestPingCommand(t *testing.T) {
	withSilencedStdout(t, func() {
		if err := cmdPing([]string{"--node", "test"}); err != nil {
			t.Fatalf("cmdPing: %v", err)
		}
	})
}
