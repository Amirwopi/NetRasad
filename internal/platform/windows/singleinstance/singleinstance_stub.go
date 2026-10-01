//go:build !windows

package singleinstance

func CheckAndLock() bool {
	return false
}
