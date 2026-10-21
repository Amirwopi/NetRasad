//go:build linux

package autostart

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuoteExec(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/usr/bin/NetRasad", "/usr/bin/NetRasad"},
		{"/opt/Net Rasad/NetRasad", `"/opt/Net Rasad/NetRasad"`},
		{`/home/u/a"b$c` + "`d", `"/home/u/a\\"b\\$c\\` + "`d\""},
		{`/home/u/back\slash`, `"/home/u/back\\\\slash"`},
		{"/home/u/100%/NetRasad", "/home/u/100%%/NetRasad"},
	}
	for _, tt := range tests {
		if got := quoteExec(tt.path); got != tt.want {
			t.Errorf("quoteExec(%q) = %s, want %s", tt.path, got, tt.want)
		}
	}
}

func TestSetAutoStartRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPIMAGE", "")

	if IsAutoStartEnabled() {
		t.Fatal("enabled before anything was written")
	}
	if err := SetAutoStart(true); err != nil {
		t.Fatal(err)
	}
	if !IsAutoStartEnabled() {
		t.Fatal("not enabled after SetAutoStart(true)")
	}

	path, _ := autostartFile()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	exe, _ := os.Executable()
	if !strings.Contains(string(data), "\nExec="+quoteExec(exe)+"\n") {
		t.Errorf("entry does not launch %s:\n%s", exe, data)
	}
	if validator, err := exec.LookPath("desktop-file-validate"); err == nil {
		if out, err := exec.Command(validator, path).CombinedOutput(); err != nil {
			t.Errorf("desktop-file-validate: %v\n%s", err, out)
		}
	}

	if err := SetAutoStart(false); err != nil {
		t.Fatal(err)
	}
	if IsAutoStartEnabled() {
		t.Fatal("still enabled after SetAutoStart(false)")
	}
	if err := SetAutoStart(false); err != nil {
		t.Fatalf("disabling twice must not fail: %v", err)
	}
}

func TestAutoStartRespectsDisabledEntries(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, _ := autostartFile()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"Hidden=true", "X-GNOME-Autostart-enabled=false"} {
		if err := os.WriteFile(path, []byte(desktopEntry("/usr/bin/NetRasad")+line+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if IsAutoStartEnabled() {
			t.Errorf("entry with %s reported as enabled", line)
		}
	}
}

func TestAutoStartLaunchesAppImage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPIMAGE", "/home/u/Apps/NetRasad.AppImage")
	if err := SetAutoStart(true); err != nil {
		t.Fatal(err)
	}
	path, _ := autostartFile()
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "\nExec=/home/u/Apps/NetRasad.AppImage\n") {
		t.Errorf("AppImage entry:\n%s", data)
	}
}
