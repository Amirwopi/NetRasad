//go:build !windows && !linux

package singleinstance

func CheckAndLock() bool {
	return false
}
