//go:build windows

package singleinstance

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modKernel32      = windows.NewLazySystemDLL("kernel32.dll")
	procCreateMutexW = modKernel32.NewProc("CreateMutexW")
)

const ERROR_ALREADY_EXISTS = 183

var mutexHandle uintptr

func CheckAndLock() bool {
	mutexName, err := syscall.UTF16PtrFromString("Local\\NetRasadSingleInstanceMutex_7B3A9")
	if err != nil {
		return false
	}

	h, _, errCall := procCreateMutexW.Call(
		0,
		1,
		uintptr(unsafe.Pointer(mutexName)),
	)

	if h == 0 {
		return false
	}

	mutexHandle = h
	if errCall != nil && errCall.(syscall.Errno) == ERROR_ALREADY_EXISTS {
		FocusExistingInstance()
		return true
	}

	return false
}

func FocusExistingInstance() {
	modUser32 := windows.NewLazySystemDLL("user32.dll")
	procFindWindowW := modUser32.NewProc("FindWindowW")
	procShowWindow := modUser32.NewProc("ShowWindow")
	procSetForegroundWindow := modUser32.NewProc("SetForegroundWindow")

	titlePtr, err := syscall.UTF16PtrFromString("NetRasad")
	if err != nil {
		return
	}

	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd != 0 {
		procShowWindow.Call(hwnd, 9)
		procSetForegroundWindow.Call(hwnd)
	}
}
