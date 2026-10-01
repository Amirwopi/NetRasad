//go:build !windows

package autostart

func IsAutoStartEnabled() bool {
	return false
}

func SetAutoStart(enable bool) error {
	return nil
}
