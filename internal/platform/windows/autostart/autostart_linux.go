//go:build linux

package autostart

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Desktop sessions start every entry in $XDG_CONFIG_HOME/autostart at login.
const desktopFileName = "netrasad.desktop"

func autostartFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", desktopFileName), nil
}

func IsAutoStartEnabled() bool {
	path, err := autostartFile()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	// Session settings tools disable entries in place instead of deleting them.
	for _, line := range strings.Split(string(data), "\n") {
		switch strings.TrimSpace(line) {
		case "Hidden=true", "X-GNOME-Autostart-enabled=false":
			return false
		}
	}
	return true
}

func SetAutoStart(enable bool) error {
	path, err := autostartFile()
	if err != nil {
		return err
	}
	if !enable {
		err := os.Remove(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	exePath, err := executablePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(desktopEntry(exePath)), 0644)
}

// executablePath returns what to launch at login. An AppImage runs from a
// temporary mount, so the image file itself has to be started instead.
func executablePath() (string, error) {
	if appImage := os.Getenv("APPIMAGE"); appImage != "" {
		return appImage, nil
	}
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(exePath, " (deleted)"), nil
}

func desktopEntry(exePath string) string {
	return "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=NetRasad\n" +
		"Comment=Network traffic monitor\n" +
		"Exec=" + quoteExec(exePath) + "\n" +
		"Icon=netrasad\n" +
		"Terminal=false\n" +
		"X-GNOME-Autostart-enabled=true\n"
}

// quoteExec encodes a path for an Exec key. Two escaping layers apply: the
// argument is double-quoted with ", `, $ and \ backslash-escaped, then the
// value is escaped as a desktop entry string, doubling every backslash.
func quoteExec(path string) string {
	path = strings.ReplaceAll(path, "%", "%%")
	if !strings.ContainsAny(path, " \t\n\r\"'\\><~|&;$*?#()`") {
		return path
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range path {
		switch r {
		case '\\':
			b.WriteString(`\\\\`)
		case '"', '`', '$':
			b.WriteString(`\\`)
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
