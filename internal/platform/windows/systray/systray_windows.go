//go:build windows

package systray

import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32               = windows.NewLazySystemDLL("user32.dll")
	modShell32              = windows.NewLazySystemDLL("shell32.dll")

	procRegisterClassExW    = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW     = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow       = modUser32.NewProc("DestroyWindow")
	procDefWindowProcW      = modUser32.NewProc("DefWindowProcW")
	procShowWindow          = modUser32.NewProc("ShowWindow")
	procSetForegroundWindow = modUser32.NewProc("SetForegroundWindow")
	procCreatePopupMenu     = modUser32.NewProc("CreatePopupMenu")
	procAppendMenuW         = modUser32.NewProc("AppendMenuW")
	procTrackPopupMenu      = modUser32.NewProc("TrackPopupMenu")
	procGetCursorPos        = modUser32.NewProc("GetCursorPos")

	procShellNotifyIconW    = modShell32.NewProc("Shell_NotifyIconW")
	procExtractIconW        = modShell32.NewProc("ExtractIconW")
	procLoadIconW           = modUser32.NewProc("LoadIconW")
	procPostMessageW        = modUser32.NewProc("PostMessageW")
	procPostQuitMessage     = modUser32.NewProc("PostQuitMessage")
)

const (
	NIM_ADD        = 0x00000000
	NIM_MODIFY     = 0x00000001
	NIM_DELETE     = 0x00000002

	NIF_MESSAGE    = 0x00000001
	NIF_ICON       = 0x00000002
	NIF_TIP        = 0x00000004

	WM_DESTROY     = 0x0002
	WM_CLOSE       = 0x0010
	WM_USER        = 0x0400
	WM_TRAYICON    = WM_USER + 1
	WM_COMMAND     = 0x0111
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP   = 0x0205

	MF_STRING      = 0x00000000
	MF_SEPARATOR   = 0x00000800

	TPM_RIGHTBUTTON = 0x0002

	IDM_OPEN       = 1001
	IDM_TOGGLEBAND = 1002
	IDM_EXIT       = 1003
)

type notifyIconDataW struct {
	CbSize           uint32
	_                uint32
	Hwnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	_                uint32
	HIcon            uintptr
	SzTip            [128]uint16
}

type point struct {
	X, Y int32
}

type SysTray struct {
	mu           sync.Mutex
	hwnd         uintptr
	hIcon        uintptr
	onOpen       func()
	onToggleBand func()
	onExit       func()
}

var globalTray *SysTray
var globalTrayOnce sync.Once

func GetSysTray() *SysTray {
	globalTrayOnce.Do(func() {
		globalTray = &SysTray{}
	})
	return globalTray
}

func (t *SysTray) Start(onOpen, onToggleBand, onExit func()) error {
	t.mu.Lock()
	if t.hwnd != 0 {
		t.mu.Unlock()
		return nil
	}
	t.onOpen = onOpen
	t.onToggleBand = onToggleBand
	t.onExit = onExit
	t.mu.Unlock()

	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		className, err := syscall.UTF16PtrFromString("NetRasadSysTrayClass")
		if err != nil {
			ready <- err
			return
		}

		wndProc := syscall.NewCallback(trayWindowProc)

		var wc struct {
			Size       uint32
			Style      uint32
			WndProc    uintptr
			ClsExtra   int32
			WndExtra   int32
			Instance   uintptr
			Icon       uintptr
			Cursor     uintptr
			Background uintptr
			MenuName   *uint16
			ClassName  *uint16
			IconSm     uintptr
		}
		wc.Size = uint32(unsafe.Sizeof(wc))
		wc.WndProc = wndProc
		wc.ClassName = className

		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

		hwnd, _, _ := procCreateWindowExW.Call(
			0, uintptr(unsafe.Pointer(className)), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		)

		if hwnd == 0 {
			ready <- nil
			return
		}

		var hIcon uintptr
		exePath, err := os.Executable()
		if err == nil {
			exePathPtr, _ := syscall.UTF16PtrFromString(exePath)
			hIcon, _, _ = procExtractIconW.Call(0, uintptr(unsafe.Pointer(exePathPtr)), 0)
		}
		if hIcon == 0 || hIcon == 1 {
			hIcon, _, _ = procLoadIconW.Call(0, 32512)
		}

		var nid notifyIconDataW
		nid.CbSize = uint32(unsafe.Sizeof(nid))
		nid.Hwnd = hwnd
		nid.UID = 1
		nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
		nid.UCallbackMessage = WM_TRAYICON
		nid.HIcon = hIcon

		tipStr := "NetRasad Network Monitor"
		tipUTF16, _ := syscall.UTF16FromString(tipStr)
		copy(nid.SzTip[:], tipUTF16)

		procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))

		t.mu.Lock()
		t.hwnd = hwnd
		t.hIcon = hIcon
		t.mu.Unlock()

		ready <- nil

		var msg struct {
			Hwnd    uintptr
			Message uint32
			WParam  uintptr
			LParam  uintptr
			Time    uint32
			Pt      point
		}

		procGetMessage := modUser32.NewProc("GetMessageW")
		procDispatchMessage := modUser32.NewProc("DispatchMessageW")

		for {
			r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if r == 0 || r == uintptr(syscall.InvalidHandle) {
				break
			}
			procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}()

	return <-ready
}

func (t *SysTray) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.hwnd != 0 {
		var nid notifyIconDataW
		nid.CbSize = uint32(unsafe.Sizeof(nid))
		nid.Hwnd = t.hwnd
		nid.UID = 1
		procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))

		procPostMessageW.Call(t.hwnd, WM_CLOSE, 0, 0)
		t.hwnd = 0
	}
}

func trayWindowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	t := GetSysTray()

	switch msg {
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	case WM_TRAYICON:
		switch lParam {
		case WM_LBUTTONDBLCLK:
			t.mu.Lock()
			fn := t.onOpen
			t.mu.Unlock()
			if fn != nil {
				fn()
			}
		case WM_RBUTTONUP:
			showTrayContextMenu(hwnd)
		}
		return 0
	case WM_COMMAND:
		cmdID := uint16(wParam & 0xFFFF)
		t.mu.Lock()
		onOpen := t.onOpen
		onToggleBand := t.onToggleBand
		onExit := t.onExit
		t.mu.Unlock()

		switch cmdID {
		case IDM_OPEN:
			if onOpen != nil {
				onOpen()
			}
		case IDM_TOGGLEBAND:
			if onToggleBand != nil {
				onToggleBand()
			}
		case IDM_EXIT:
			if onExit != nil {
				onExit()
			}
		}
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func showTrayContextMenu(hwnd uintptr) {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}

	strOpen, _ := syscall.UTF16PtrFromString("Open NetRasad")
	strToggle, _ := syscall.UTF16PtrFromString("Toggle Speed Band")
	strExit, _ := syscall.UTF16PtrFromString("Exit NetRasad")

	procAppendMenuW.Call(hMenu, MF_STRING, IDM_OPEN, uintptr(unsafe.Pointer(strOpen)))
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_TOGGLEBAND, uintptr(unsafe.Pointer(strToggle)))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, IDM_EXIT, uintptr(unsafe.Pointer(strExit)))

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	procSetForegroundWindow.Call(hwnd)
	procTrackPopupMenu.Call(hMenu, TPM_RIGHTBUTTON, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
}
