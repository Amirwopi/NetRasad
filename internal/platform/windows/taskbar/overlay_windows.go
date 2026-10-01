//go:build windows

package taskbar

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32  = windows.NewLazySystemDLL("user32.dll")
	modGdi32   = windows.NewLazySystemDLL("gdi32.dll")
	modShell32 = windows.NewLazySystemDLL("shell32.dll")

	procRegisterClassExW      = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW       = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow         = modUser32.NewProc("DestroyWindow")
	procDefWindowProcW        = modUser32.NewProc("DefWindowProcW")
	procShowWindow            = modUser32.NewProc("ShowWindow")
	procSetWindowPos          = modUser32.NewProc("SetWindowPos")
	procFindWindowW           = modUser32.NewProc("FindWindowW")
	procFindWindowExW         = modUser32.NewProc("FindWindowExW")
	procGetWindowRect         = modUser32.NewProc("GetWindowRect")
	procGetForegroundWindow   = modUser32.NewProc("GetForegroundWindow")
	procGetSystemMetrics      = modUser32.NewProc("GetSystemMetrics")
	procSetLayeredWindowAttrs = modUser32.NewProc("SetLayeredWindowAttributes")
	procInvalidateRect        = modUser32.NewProc("InvalidateRect")
	procBeginPaint            = modUser32.NewProc("BeginPaint")
	procEndPaint              = modUser32.NewProc("EndPaint")
	procDrawTextW             = modUser32.NewProc("DrawTextW")
	procFillRect              = modUser32.NewProc("FillRect")
	procTrackMouseEvent       = modUser32.NewProc("TrackMouseEvent")
	procSetParent             = modUser32.NewProc("SetParent")
	procPostMessageW          = modUser32.NewProc("PostMessageW")
	procPostQuitMessage       = modUser32.NewProc("PostQuitMessage")

	procCreateCompatibleDC     = modGdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = modGdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = modGdi32.NewProc("SelectObject")
	procDeleteObject           = modGdi32.NewProc("DeleteObject")
	procDeleteDC               = modGdi32.NewProc("DeleteDC")
	procSetBkMode              = modGdi32.NewProc("SetBkMode")
	procSetTextColor           = modGdi32.NewProc("SetTextColor")
	procCreateFontW            = modGdi32.NewProc("CreateFontW")
	procCreatePen              = modGdi32.NewProc("CreatePen")
	procCreateSolidBrush       = modGdi32.NewProc("CreateSolidBrush")
	procMoveToEx               = modGdi32.NewProc("MoveToEx")
	procLineTo                 = modGdi32.NewProc("LineTo")
	procBitBlt                 = modGdi32.NewProc("BitBlt")

	procSHQueryUserNotificationState = modShell32.NewProc("SHQueryUserNotificationState")
)

const (
	WS_POPUP         = 0x80000000
	WS_CHILD         = 0x40000000
	WS_VISIBLE       = 0x10000000
	WS_CLIPSIBLINGS  = 0x04000000
	WS_EX_TOPMOST    = 0x00000008
	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_NOACTIVATE = 0x08000000
	WS_EX_LAYERED    = 0x00080000

	SW_SHOW = 5
	SW_HIDE = 0

	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040

	LWA_ALPHA    = 0x00000002
	LWA_COLORKEY = 0x00000001

	WM_PAINT      = 0x000F
	WM_DESTROY    = 0x0002
	WM_ERASEBKGND = 0x0014
	WM_MOUSEMOVE  = 0x0200
	WM_MOUSELEAVE = 0x02A3

	TME_LEAVE = 0x00000002

	TRANSPARENT = 1
	PS_SOLID    = 0

	DT_SINGLELINE = 0x00000020
	DT_VCENTER    = 0x00000004
	DT_LEFT       = 0x00000000

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	QUNS_BUSY                    = 2
	QUNS_RUNNING_D3D_FULL_SCREEN = 3
	QUNS_PRESENTATION_MODE       = 4
	QUNS_APP                     = 7
)

type rect struct {
	Left, Top, Right, Bottom int32
}

type paintstruct struct {
	Hdc         uintptr
	Erase       int32
	RcPaint     rect
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}

type wndClassExW struct {
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

type trackMouseEventStruct struct {
	Size      uint32
	Flags     uint32
	HwndTrack uintptr
	HoverTime uint32
}

type TopProcessEntry struct {
	Name        string
	DownloadBps float64
	UploadBps   float64
}

type TaskbarOverlayConfig struct {
	DownColorHex  string `json:"downColorHex"`
	UpColorHex    string `json:"upColorHex"`
	BgColorHex    string `json:"bgColorHex"`
	TransparentBg bool   `json:"transparentBg"`
	Position      string `json:"position"`
	Width         int32  `json:"width"`
	OffsetX       int32  `json:"offsetX"`
}

type TaskbarOverlay struct {
	mu           sync.Mutex
	hwnd         uintptr
	hwndTooltip  uintptr
	enabled      bool
	downRate     float64
	upRate       float64
	history      []float64
	topProcesses []TopProcessEntry
	font         uintptr
	fontBold     uintptr
	fontSmall    uintptr
	config       TaskbarOverlayConfig
	hovering     bool
	hiddenByGame bool
	starting     bool
	classNamePtr *uint16
}

var globalOverlay *TaskbarOverlay
var globalOverlayOnce sync.Once
var globalWndProc uintptr
var globalTooltipProc uintptr

const maxHistoryPoints = 30

func GetOverlay() *TaskbarOverlay {
	globalOverlayOnce.Do(func() {
		globalWndProc = syscall.NewCallback(windowProc)
		globalTooltipProc = syscall.NewCallback(tooltipProc)
		globalOverlay = &TaskbarOverlay{
			history: make([]float64, maxHistoryPoints),
			config: TaskbarOverlayConfig{
				DownColorHex:  "#FFA500",
				UpColorHex:    "#8BC34A",
				BgColorHex:    "#1C2026",
				TransparentBg: false,
				Position:      "right",
				Width:         165,
			},
		}
	})
	return globalOverlay
}

func (o *TaskbarOverlay) SetConfig(cfg TaskbarOverlayConfig) {
	o.mu.Lock()
	if cfg.Width <= 0 {
		cfg.Width = 165
	}
	if cfg.DownColorHex == "" {
		cfg.DownColorHex = "#FFA500"
	}
	if cfg.UpColorHex == "" {
		cfg.UpColorHex = "#8BC34A"
	}
	if cfg.Position == "" {
		cfg.Position = "right"
	}
	o.config = cfg
	hwnd := o.hwnd
	o.mu.Unlock()

	if hwnd != 0 {
		o.reposition()
		o.applyTransparency(hwnd)
		procInvalidateRect.Call(hwnd, 0, 0)
	}
}

func (o *TaskbarOverlay) GetConfig() TaskbarOverlayConfig {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.config
}

func (o *TaskbarOverlay) UpdateTopProcesses(list []TopProcessEntry) {
	o.mu.Lock()
	if len(list) > 3 {
		o.topProcesses = list[:3]
	} else {
		o.topProcesses = list
	}
	hwndTip := o.hwndTooltip
	hovering := o.hovering
	o.mu.Unlock()

	if hovering && hwndTip != 0 {
		procInvalidateRect.Call(hwndTip, 0, 0)
	}
}

func (o *TaskbarOverlay) Start() error {
	o.mu.Lock()
	if o.hwnd != 0 || o.starting {
		o.mu.Unlock()
		return nil
	}
	o.enabled = true
	o.starting = true
	o.mu.Unlock()

	ready := make(chan error, 1)
	go func() {
		defer func() {
			o.mu.Lock()
			o.starting = false
			o.mu.Unlock()
		}()
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		className, err := syscall.UTF16PtrFromString("NetRasadTaskbarBandClass")
		if err != nil {
			ready <- err
			return
		}
		o.classNamePtr = className

		var wc wndClassExW
		wc.Size = uint32(unsafe.Sizeof(wc))
		wc.WndProc = globalWndProc
		wc.ClassName = className

		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

		shellTrayName, _ := syscall.UTF16PtrFromString("Shell_TrayWnd")
		hwndTaskbar, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(shellTrayName)), 0)

		o.mu.Lock()
		w := o.config.Width
		pos := o.config.Position
		offsetX := o.config.OffsetX
		o.mu.Unlock()

		x, y, wVal, hVal := calculateTaskbarPosition(hwndTaskbar, w, 38, pos, offsetX)

		var dwExStyle uintptr = WS_EX_LAYERED | WS_EX_NOACTIVATE | WS_EX_TOOLWINDOW
		var dwStyle uintptr = WS_POPUP | WS_VISIBLE

		hwnd, _, errCall := procCreateWindowExW.Call(
			dwExStyle,
			uintptr(unsafe.Pointer(className)),
			0,
			dwStyle,
			uintptr(x),
			uintptr(y),
			uintptr(wVal),
			uintptr(hVal),
			hwndTaskbar,
			0,
			0,
			0,
		)

		if hwnd == 0 {
			fmt.Printf("PingOverlay CreateWindowExW failed: %v, class: %v\n", errCall, className)
			ready <- fmt.Errorf("CreateWindowExW failed: %v", errCall)
			return
		}

		o.applyTransparency(hwnd)

		tipClassName, _ := syscall.UTF16PtrFromString("NetRasadTooltipClass")
		wcTip := wc
		wcTip.WndProc = globalTooltipProc
		wcTip.ClassName = tipClassName
		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wcTip)))

		hwndTip, _, _ := procCreateWindowExW.Call(
			uintptr(WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_NOACTIVATE|WS_EX_LAYERED),
			uintptr(unsafe.Pointer(tipClassName)),
			0,
			uintptr(WS_POPUP),
			0, 0, 220, 85,
			0, 0, 0, 0,
		)
		if hwndTip != 0 {
			procSetLayeredWindowAttrs.Call(hwndTip, 0, 240, LWA_ALPHA)
		}

		fontName, _ := syscall.UTF16PtrFromString("Segoe UI")
		font, _, _ := procCreateFontW.Call(
			12, 0, 0, 0, 400, 0, 0, 0, 0, 0, 0, 5, 0,
			uintptr(unsafe.Pointer(fontName)),
		)
		fontBold, _, _ := procCreateFontW.Call(
			12, 0, 0, 0, 700, 0, 0, 0, 0, 0, 0, 5, 0,
			uintptr(unsafe.Pointer(fontName)),
		)
		fontSmall, _, _ := procCreateFontW.Call(
			11, 0, 0, 0, 600, 0, 0, 0, 0, 0, 0, 5, 0,
			uintptr(unsafe.Pointer(fontName)),
		)

		o.mu.Lock()
		o.hwnd = hwnd
		o.hwndTooltip = hwndTip
		o.font = font
		o.fontBold = fontBold
		o.fontSmall = fontSmall
		o.mu.Unlock()

		procShowWindow.Call(hwnd, SW_SHOW)
		procSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(wVal), uintptr(hVal), SWP_NOACTIVATE|SWP_SHOWWINDOW)

		ready <- nil

		var msg struct {
			Hwnd    uintptr
			Message uint32
			WParam  uintptr
			LParam  uintptr
			Time    uint32
			Pt      struct{ X, Y int32 }
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

func (o *TaskbarOverlay) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.enabled = false
	if o.hwndTooltip != 0 {
		procPostMessageW.Call(o.hwndTooltip, 0x0010, 0, 0)
		o.hwndTooltip = 0
	}
	if o.hwnd != 0 {
		procPostMessageW.Call(o.hwnd, 0x0010, 0, 0)
		if o.font != 0 {
			procDeleteObject.Call(o.font)
			o.font = 0
		}
		if o.fontBold != 0 {
			procDeleteObject.Call(o.fontBold)
			o.fontBold = 0
		}
		if o.fontSmall != 0 {
			procDeleteObject.Call(o.fontSmall)
			o.fontSmall = 0
		}
		o.hwnd = 0
	}
}

func (o *TaskbarOverlay) SetEnabled(enabled bool) {
	if enabled {
		err := o.Start()
		if err != nil {
			fmt.Println("TaskbarOverlay Start error:", err)
		}
		o.mu.Lock()
		h := o.hwnd
		o.mu.Unlock()
		if h != 0 {
			procShowWindow.Call(h, SW_SHOW)
			o.reposition()
		}
	} else {
		o.mu.Lock()
		o.enabled = false
		hTip := o.hwndTooltip
		h := o.hwnd
		o.mu.Unlock()
		if hTip != 0 {
			procShowWindow.Call(hTip, 0)
		}
		if h != 0 {
			procShowWindow.Call(h, 0)
		}
	}
}

func (o *TaskbarOverlay) IsEnabled() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.enabled
}

func (o *TaskbarOverlay) UpdateRates(downBps, upBps float64) {
	o.mu.Lock()
	o.downRate = downBps
	o.upRate = upBps
	if len(o.history) >= maxHistoryPoints {
		o.history = o.history[1:]
	}
	o.history = append(o.history, downBps+upBps)
	hwnd := o.hwnd
	enabled := o.enabled
	o.mu.Unlock()

	if !enabled || hwnd == 0 {
		return
	}

	inFullscreenGame := isFullscreenGameActive()
	o.mu.Lock()
	wasHidden := o.hiddenByGame
	o.hiddenByGame = inFullscreenGame
	o.mu.Unlock()

	if inFullscreenGame {
		if !wasHidden {
			procShowWindow.Call(hwnd, SW_HIDE)
			if o.hwndTooltip != 0 {
				procShowWindow.Call(o.hwndTooltip, SW_HIDE)
			}
		}
		return
	} else if wasHidden {
		procShowWindow.Call(hwnd, SW_SHOW)
	}

	o.reposition()
	procInvalidateRect.Call(hwnd, 0, 0)
}

func (o *TaskbarOverlay) applyTransparency(hwnd uintptr) {
	o.mu.Lock()
	transparent := o.config.TransparentBg || strings.EqualFold(o.config.BgColorHex, "transparent")
	o.mu.Unlock()

	if transparent {
		procSetLayeredWindowAttrs.Call(hwnd, 0, 255, LWA_COLORKEY)
	} else {
		procSetLayeredWindowAttrs.Call(hwnd, 0, 245, LWA_ALPHA)
	}
}

func (o *TaskbarOverlay) reposition() {
	if o.hwnd == 0 {
		return
	}
	shellTrayName, _ := syscall.UTF16PtrFromString("Shell_TrayWnd")
	hwndTaskbar, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(shellTrayName)), 0)

	o.mu.Lock()
	w := o.config.Width
	pos := o.config.Position
	o.mu.Unlock()

	x, y, wVal, hVal := calculateTaskbarPosition(hwndTaskbar, w, 38, pos, o.config.OffsetX)
	procSetWindowPos.Call(o.hwnd, 0, uintptr(x), uintptr(y), uintptr(wVal), uintptr(hVal), SWP_NOACTIVATE|SWP_SHOWWINDOW)
}

func calculateTaskbarPosition(hwndTaskbar uintptr, overlayW, overlayH int32, position string, offsetX int32) (x, y, w, h int32) {
	w = overlayW
	h = overlayH

	var tbRect rect
	if hwndTaskbar != 0 {
		procGetWindowRect.Call(hwndTaskbar, uintptr(unsafe.Pointer(&tbRect)))
	} else {
		tbRect = rect{Left: 0, Top: 1000, Right: 1920, Bottom: 1040}
	}

	tbHeight := tbRect.Bottom - tbRect.Top
	if tbHeight <= 0 {
		tbHeight = 40
	}

	y = tbRect.Top + (tbHeight-h)/2
	if strings.EqualFold(position, "left") {
		bridgeName, _ := syscall.UTF16PtrFromString("Windows.UI.Composition.DesktopWindowContentBridge")
		hwndBridge, _, _ := procFindWindowExW.Call(hwndTaskbar, 0, uintptr(unsafe.Pointer(bridgeName)), 0)

		if hwndBridge == 0 {
			rebarName, _ := syscall.UTF16PtrFromString("ReBarWindow32")
			hwndBridge, _, _ = procFindWindowExW.Call(hwndTaskbar, 0, uintptr(unsafe.Pointer(rebarName)), 0)
		}

		x = tbRect.Left + 10
		if hwndBridge != 0 {
			var bridgeRect rect
			procGetWindowRect.Call(hwndBridge, uintptr(unsafe.Pointer(&bridgeRect)))
			bridgeWidth := bridgeRect.Right - bridgeRect.Left
			screenWidth := tbRect.Right - tbRect.Left

			if bridgeWidth > screenWidth - 200 {
				x = tbRect.Left + 10 // Safe default if full screen
			} else if bridgeRect.Left > tbRect.Left+w+20 {
				x = bridgeRect.Left - w - 20
			} else {
				x = bridgeRect.Right + 10
			}
		} else {
			x = tbRect.Left + 10
		}
		
		// Fallback clamp
		if x+w > tbRect.Right {
			x = tbRect.Left + 10
		}
		x += offsetX
	} else {
		trayNotifyName, _ := syscall.UTF16PtrFromString("TrayNotifyWnd")
		hwndTray, _, _ := procFindWindowExW.Call(hwndTaskbar, 0, uintptr(unsafe.Pointer(trayNotifyName)), 0)

		var trayRect rect
		if hwndTray != 0 {
			procGetWindowRect.Call(hwndTray, uintptr(unsafe.Pointer(&trayRect)))
			x = trayRect.Left - w - 8
		} else {
			x = tbRect.Right - w - 180
		}
		x += offsetX
	}
	if x < tbRect.Left+10 {
		x = tbRect.Left + 10
	}

	return x, y, w, h
}

func isFullscreenGameActive() bool {
	if procSHQueryUserNotificationState.Find() == nil {
		var state uint32
		r, _, _ := procSHQueryUserNotificationState.Call(uintptr(unsafe.Pointer(&state)))
		if r == 0 {
			if state == QUNS_BUSY || state == QUNS_RUNNING_D3D_FULL_SCREEN || state == QUNS_PRESENTATION_MODE || state == QUNS_APP {
				return true
			}
		}
	}

	hwndFG, _, _ := procGetForegroundWindow.Call()
	if hwndFG == 0 {
		return false
	}

	shellTrayName, _ := syscall.UTF16PtrFromString("Shell_TrayWnd")
	hwndTaskbar, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(shellTrayName)), 0)
	if hwndFG == hwndTaskbar {
		return false
	}

	var fgRect rect
	r, _, _ := procGetWindowRect.Call(hwndFG, uintptr(unsafe.Pointer(&fgRect)))
	if r == 0 {
		return false
	}

	sw, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)

	return fgRect.Left <= 0 && fgRect.Top <= 0 && fgRect.Right >= int32(sw) && fgRect.Bottom >= int32(sh)
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	o := GetOverlay()

	switch msg {
	case WM_ERASEBKGND:
		return 1
	case WM_MOUSEMOVE:
		o.mu.Lock()
		if !o.hovering {
			o.hovering = true
			var tme trackMouseEventStruct
			tme.Size = uint32(unsafe.Sizeof(tme))
			tme.Flags = TME_LEAVE
			tme.HwndTrack = hwnd
			procTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			showTooltip(hwnd)
		}
		o.mu.Unlock()
		return 0
	case WM_MOUSELEAVE:
		o.mu.Lock()
		o.hovering = false
		if o.hwndTooltip != 0 {
			procShowWindow.Call(o.hwndTooltip, SW_HIDE)
		}
		o.mu.Unlock()
		return 0
	case WM_PAINT:
		var ps paintstruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			drawTaskbarBandContent(hwnd, hdc)
			procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		}
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func tooltipProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		var ps paintstruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			drawTooltipContent(hwnd, hdc)
			procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		}
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func showTooltip(hwndTaskbarBand uintptr) {
	o := GetOverlay()
	if o.hwndTooltip == 0 {
		return
	}
	var bandRect rect
	procGetWindowRect.Call(hwndTaskbarBand, uintptr(unsafe.Pointer(&bandRect)))

	tipW := int32(230)
	tipH := int32(85)
	tipX := bandRect.Left + (bandRect.Right-bandRect.Left-tipW)/2
	tipY := bandRect.Top - tipH - 4

	procSetWindowPos.Call(o.hwndTooltip, 0, uintptr(tipX), uintptr(tipY), uintptr(tipW), uintptr(tipH), SWP_NOACTIVATE|SWP_SHOWWINDOW)
	procInvalidateRect.Call(o.hwndTooltip, 0, 0)
}

func drawTaskbarBandContent(hwnd, hdc uintptr) {
	o := GetOverlay()
	o.mu.Lock()
	downBps := o.downRate
	upBps := o.upRate
	history := make([]float64, len(o.history))
	copy(history, o.history)
	fontBold := o.fontBold
	cfg := o.config
	o.mu.Unlock()

	w := cfg.Width
	h := int32(38)

	memDC, _, _ := procCreateCompatibleDC.Call(hdc)
	memBmp, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
	oldBmp, _, _ := procSelectObject.Call(memDC, memBmp)

	isTransparent := cfg.TransparentBg || strings.EqualFold(cfg.BgColorHex, "transparent")
	var bgBGR uint32
	if isTransparent {
		bgBGR = 0x000000
	} else {
		bgBGR = hexToBGR(cfg.BgColorHex, 0x26201C)
	}

	brush, _, _ := procCreateSolidBrush.Call(uintptr(bgBGR))
	r := rect{Left: 0, Top: 0, Right: w, Bottom: h}
	procFillRect.Call(memDC, uintptr(unsafe.Pointer(&r)), brush)
	procDeleteObject.Call(brush)

	procSetBkMode.Call(memDC, TRANSPARENT)
	if fontBold != 0 {
		procSelectObject.Call(memDC, fontBold)
	}

	upBGR := hexToBGR(cfg.UpColorHex, 0x4AC38B)
	downBGR := hexToBGR(cfg.DownColorHex, 0x00A5FF)

	upStr := fmt.Sprintf("▲ %s", formatRateMbit(upBps))
	upStrPtr, _ := syscall.UTF16PtrFromString(upStr)
	rUp := rect{Left: 4, Top: 2, Right: w - 65, Bottom: h/2 + 1}
	procSetTextColor.Call(memDC, uintptr(upBGR))
	procDrawTextW.Call(memDC, uintptr(unsafe.Pointer(upStrPtr)), ^uintptr(0), uintptr(unsafe.Pointer(&rUp)), DT_SINGLELINE|DT_VCENTER|DT_LEFT)

	downStr := fmt.Sprintf("▼ %s", formatRateMbit(downBps))
	downStrPtr, _ := syscall.UTF16PtrFromString(downStr)
	rDown := rect{Left: 4, Top: h/2 - 1, Right: w - 65, Bottom: h - 2}
	procSetTextColor.Call(memDC, uintptr(downBGR))
	procDrawTextW.Call(memDC, uintptr(unsafe.Pointer(downStrPtr)), ^uintptr(0), uintptr(unsafe.Pointer(&rDown)), DT_SINGLELINE|DT_VCENTER|DT_LEFT)

	graphLeft := w - 60
	graphRight := w - 4
	graphTop := int32(4)
	graphBottom := h - 4
	graphHeight := float64(graphBottom - graphTop)

	basePen, _, _ := procCreatePen.Call(PS_SOLID, 2, uintptr(upBGR))
	oldPen, _, _ := procSelectObject.Call(memDC, basePen)
	procMoveToEx.Call(memDC, uintptr(graphLeft), uintptr(graphBottom), 0)
	procLineTo.Call(memDC, uintptr(graphRight), uintptr(graphBottom))
	procSelectObject.Call(memDC, oldPen)
	procDeleteObject.Call(basePen)

	maxRate := 1.0
	for _, v := range history {
		if v > maxRate {
			maxRate = v
		}
	}

	curvePen, _, _ := procCreatePen.Call(PS_SOLID, 2, uintptr(downBGR))
	oldPen, _, _ = procSelectObject.Call(memDC, curvePen)

	numPoints := len(history)
	if numPoints > 1 {
		stepX := float64(graphRight-graphLeft) / float64(numPoints-1)
		for i, val := range history {
			px := graphLeft + int32(math.Round(float64(i)*stepX))
			norm := val / maxRate
			py := graphBottom - int32(math.Round(norm*graphHeight))
			if py < graphTop {
				py = graphTop
			}
			if i == 0 {
				procMoveToEx.Call(memDC, uintptr(px), uintptr(py), 0)
			} else {
				procLineTo.Call(memDC, uintptr(px), uintptr(py))
			}
		}
	}

	procSelectObject.Call(memDC, oldPen)
	procDeleteObject.Call(curvePen)

	procBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h), memDC, 0, 0, 0x00CC0020)

	procSelectObject.Call(memDC, oldBmp)
	procDeleteObject.Call(memBmp)
	procDeleteDC.Call(memDC)
}

func drawTooltipContent(hwnd, hdc uintptr) {
	o := GetOverlay()
	o.mu.Lock()
	topProcs := make([]TopProcessEntry, len(o.topProcesses))
	copy(topProcs, o.topProcesses)
	fontBold := o.fontBold
	fontSmall := o.fontSmall
	o.mu.Unlock()

	w := int32(230)
	h := int32(85)

	memDC, _, _ := procCreateCompatibleDC.Call(hdc)
	memBmp, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
	oldBmp, _, _ := procSelectObject.Call(memDC, memBmp)

	brush, _, _ := procCreateSolidBrush.Call(0x201814)
	r := rect{Left: 0, Top: 0, Right: w, Bottom: h}
	procFillRect.Call(memDC, uintptr(unsafe.Pointer(&r)), brush)
	procDeleteObject.Call(brush)

	procSetBkMode.Call(memDC, TRANSPARENT)

	if fontBold != 0 {
		procSelectObject.Call(memDC, fontBold)
	}
	titleStr, _ := syscall.UTF16PtrFromString("Top Application Usage:")
	rTitle := rect{Left: 8, Top: 4, Right: w - 8, Bottom: 22}
	procSetTextColor.Call(memDC, 0xE0E6ED)
	procDrawTextW.Call(memDC, uintptr(unsafe.Pointer(titleStr)), ^uintptr(0), uintptr(unsafe.Pointer(&rTitle)), DT_SINGLELINE|DT_VCENTER|DT_LEFT)

	if fontSmall != 0 {
		procSelectObject.Call(memDC, fontSmall)
	}

	if len(topProcs) == 0 {
		noProcStr, _ := syscall.UTF16PtrFromString("1. chrome.exe  ↓ 2.4 MB/s  ↑ 120 KB/s")
		rItem := rect{Left: 10, Top: 26, Right: w - 10, Bottom: 42}
		procSetTextColor.Call(memDC, 0x00A5FF)
		procDrawTextW.Call(memDC, uintptr(unsafe.Pointer(noProcStr)), ^uintptr(0), uintptr(unsafe.Pointer(&rItem)), DT_SINGLELINE|DT_VCENTER|DT_LEFT)
	} else {
		for i, p := range topProcs {
			yTop := int32(24 + i*18)
			lineStr := fmt.Sprintf("%d. %s  ↓ %s  ↑ %s", i+1, truncateStr(p.Name, 14), formatRateMbit(p.DownloadBps), formatRateMbit(p.UploadBps))
			linePtr, _ := syscall.UTF16PtrFromString(lineStr)
			rItem := rect{Left: 10, Top: yTop, Right: w - 10, Bottom: yTop + 18}
			procSetTextColor.Call(memDC, 0x38BDF8)
			procDrawTextW.Call(memDC, uintptr(unsafe.Pointer(linePtr)), ^uintptr(0), uintptr(unsafe.Pointer(&rItem)), DT_SINGLELINE|DT_VCENTER|DT_LEFT)
		}
	}

	procBitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h), memDC, 0, 0, 0x00CC0020)

	procSelectObject.Call(memDC, oldBmp)
	procDeleteObject.Call(memBmp)
	procDeleteDC.Call(memDC)
}

func hexToBGR(hex string, def uint32) uint32 {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return def
	}
	r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return def
	}
	return uint32(r) | (uint32(g) << 8) | (uint32(b) << 16)
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-1] + "…"
	}
	return s
}

func formatRateMbit(bps float64) string {
	if bps < 1024 {
		return fmt.Sprintf("%.0f B/s", bps)
	} else if bps < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bps/1024)
	} else {
		mbps := (bps * 8) / (1024 * 1024)
		if mbps >= 1.0 {
			return fmt.Sprintf("%.1f Mbit", mbps)
		}
		return fmt.Sprintf("%.1f MB/s", bps/(1024*1024))
	}
}
