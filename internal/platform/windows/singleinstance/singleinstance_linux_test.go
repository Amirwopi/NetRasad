//go:build linux

package singleinstance

import (
	"bufio"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestSecondLaunchActivatesFirst(t *testing.T) {
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon not installed")
	}
	cmd := exec.Command(daemon, "--session", "--nofork", "--print-address")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start dbus-daemon: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	addr, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", strings.TrimSpace(addr))

	activated := make(chan struct{}, 1)
	OnActivate(func() { activated <- struct{}{} })
	defer OnActivate(nil)

	if CheckAndLock() {
		t.Fatal("first instance must not see another one")
	}
	defer func() {
		mu.Lock()
		conn.Close()
		conn = nil
		mu.Unlock()
	}()

	if !CheckAndLock() {
		t.Fatal("second instance must find the first")
	}
	select {
	case <-activated:
	case <-time.After(5 * time.Second):
		t.Fatal("first instance was not asked to show itself")
	}
}

func TestNoSessionBusDoesNotBlockStartup(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/netrasad-test-bus")
	if CheckAndLock() {
		t.Fatal("CheckAndLock must not report another instance without a bus")
	}
}
