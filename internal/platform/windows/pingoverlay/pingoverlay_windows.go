//go:build windows

package pingoverlay

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/netrasad/netrasad/internal/diagnostics"
	"golang.org/x/sys/windows"
)

var (
	modUser32 = windows.NewLazySystemDLL("user32.dll")
	modGdi32  = windows.NewLazySystemDLL("gdi32.dll")
	modIphlp  = windows.NewLazySystemDLL("iphlpapi.dll")

	procRegisterClassExW      = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW       = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow         = modUser32.NewProc("DestroyWindow")
	procDefWindowProcW        = modUser32.NewProc("DefWindowProcW")
	procShowWindow            = modUser32.NewProc("ShowWindow")
	procSetWindowPos          = modUser32.NewProc("SetWindowPos")
	procPostMessageW          = modUser32.NewProc("PostMessageW")
	procPostQuitMessage       = modUser32.NewProc("PostQuitMessage")
	procBeginPaint            = modUser32.NewProc("BeginPaint")
	procEndPaint              = modUser32.NewProc("EndPaint")
	procInvalidateRect        = modUser32.NewProc("InvalidateRect")
	procSetLayeredWindowAttrs = modUser32.NewProc("SetLayeredWindowAttributes")
	procDrawTextW             = modUser32.NewProc("DrawTextW")
	procReleaseCapture        = modUser32.NewProc("ReleaseCapture")
	procGetWindowRect         = modUser32.NewProc("GetWindowRect")
	procGetMessageW           = modUser32.NewProc("GetMessageW")
	procDispatchMessageW      = modUser32.NewProc("DispatchMessageW")
	procFillRect              = modUser32.NewProc("FillRect")

	procCreateCompatibleDC     = modGdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = modGdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = modGdi32.NewProc("SelectObject")
	procDeleteObject           = modGdi32.NewProc("DeleteObject")
	procDeleteDC               = modGdi32.NewProc("DeleteDC")
	procSetBkMode              = modGdi32.NewProc("SetBkMode")
	procSetTextColor           = modGdi32.NewProc("SetTextColor")
	procCreateFontW            = modGdi32.NewProc("CreateFontW")
	procCreateSolidBrush       = modGdi32.NewProc("CreateSolidBrush")
	procEllipse                = modGdi32.NewProc("Ellipse")
	procBitBlt                 = modGdi32.NewProc("BitBlt")
	procSetBkColor             = modGdi32.NewProc("SetBkColor")
	procGetStockObject         = modGdi32.NewProc("GetStockObject")

	procIcmpCreateFile  = modIphlp.NewProc("IcmpCreateFile")
	procIcmpSendEcho    = modIphlp.NewProc("IcmpSendEcho")
	procIcmpCloseHandle = modIphlp.NewProc("IcmpCloseHandle")
)

const (
	WS_POPUP         = 0x80000000
	WS_VISIBLE       = 0x10000000
	WS_EX_TOPMOST    = 0x00000008
	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_LAYERED    = 0x00080000

	SW_SHOW = 5
	SW_HIDE = 0

	SWP_NOSIZE     = 0x0001
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010

	LWA_COLORKEY = 0x00000001

	WM_PAINT       = 0x000F
	WM_DESTROY     = 0x0002
	WM_ERASEBKGND  = 0x0014
	WM_NCHITTEST   = 0x0084
	WM_CLOSE       = 0x0010
	WM_LBUTTONDOWN = 0x0201

	HTCAPTION = 2

	TRANSPARENT = 1

	DT_SINGLELINE = 0x00000020
	DT_VCENTER    = 0x00000004
	DT_CENTER     = 0x00000001
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

type Config struct {
	Enabled      bool   `json:"enabled"`
	Address      string `json:"address"`
	Label        string `json:"label"`
	X            int32  `json:"x"`
	Y            int32  `json:"y"`
	BgColorHex   string `json:"bgColorHex"`
	TextColorHex string `json:"textColorHex"`
	Shape        string `json:"shape"`
}

type PingOverlay struct {
	mu           sync.Mutex
	hwnd         uintptr
	starting     bool
	fontTitle    uintptr
	fontLabel    uintptr
	config       Config
	currentPing  int64
	classNamePtr *uint16
	stopPing     chan struct{}
}

var (
	globalOverlay     *PingOverlay
	globalOverlayOnce sync.Once
	globalWndProc     uintptr
)

func GetOverlay() *PingOverlay {
	globalOverlayOnce.Do(func() {
		globalWndProc = syscall.NewCallback(windowProc)
		globalOverlay = &PingOverlay{
			config: Config{
				Enabled:      false,
				Address:      "8.8.8.8",
				Label:        "Google",
				X:            100,
				Y:            100,
				BgColorHex:   "#26201C",
				TextColorHex: "#AAAAAA",
				Shape:        "rectangle",
			},
		}
	})
	return globalOverlay
}

func (o *PingOverlay) SetConfig(cfg Config) {
	o.mu.Lock()
	oldEnabled := o.config.Enabled
	if cfg.X == 0 && cfg.Y == 0 {
		cfg.X = o.config.X
		cfg.Y = o.config.Y
		if cfg.X == 0 {
			cfg.X = 100
			cfg.Y = 100
		}
	}
	if cfg.Address == "" {
		cfg.Address = "8.8.8.8"
	}
	if cfg.Label == "" {
		cfg.Label = "Ping"
	}
	o.config = cfg
	hwnd := o.hwnd
	o.mu.Unlock()

	if cfg.Enabled && !oldEnabled {
		o.Start()
	} else if !cfg.Enabled && oldEnabled {
		o.Stop()
	} else if cfg.Enabled && hwnd != 0 {
		winW := int32(100)
		winH := int32(44)
		if cfg.Shape == "circle" {
			winW = 64
			winH = 64
		}
		procSetWindowPos.Call(hwnd, 0, 0, 0, uintptr(winW), uintptr(winH), 0x0006)
		procInvalidateRect.Call(hwnd, 0, 0)
	}
}

func (o *PingOverlay) GetConfig() Config {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.hwnd != 0 {
		var r rect
		if r1, _, _ := procGetWindowRect.Call(o.hwnd, uintptr(unsafe.Pointer(&r))); r1 != 0 {
			o.config.X = r.Left
			o.config.Y = r.Top
		}
	}
	return o.config
}

func (o *PingOverlay) Start() error {
	o.mu.Lock()
	if o.hwnd != 0 || o.starting {
		o.mu.Unlock()
		return nil
	}
	o.starting = true
	x := o.config.X
	y := o.config.Y
	if x == 0 {
		x = 100
		y = 100
	}
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

		classStr := fmt.Sprintf("NetRasadPingOverlayClass_%d", time.Now().UnixNano())
		className, _ := syscall.UTF16PtrFromString(classStr)
		o.classNamePtr = className

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
		wc.WndProc = globalWndProc
		wc.ClassName = className

		procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

		o.mu.Lock()
		shape := o.config.Shape
		o.mu.Unlock()

		winW := int32(100)
		winH := int32(44)
		if shape == "circle" {
			winW = 64
			winH = 64
		}

		hwnd, _, errCall := procCreateWindowExW.Call(
			uintptr(WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_LAYERED),
			uintptr(unsafe.Pointer(className)),
			0,
			uintptr(WS_POPUP|WS_VISIBLE),
			uintptr(x),
			uintptr(y),
			uintptr(winW), uintptr(winH),
			0, 0, 0, 0,
		)

		if hwnd == 0 {
			fmt.Printf("PingOverlay CreateWindowExW failed: %v\n", errCall)
			ready <- fmt.Errorf("CreateWindowExW failed: %v", errCall)
			return
		}

		procSetLayeredWindowAttrs.Call(hwnd, 0x00FF00FF, 0, LWA_COLORKEY)

		fontName, _ := syscall.UTF16PtrFromString("Segoe UI")
		fontTitle, _, _ := procCreateFontW.Call(
			18, 0, 0, 0, 700, 0, 0, 0, 0, 0, 0, 4, 0,
			uintptr(unsafe.Pointer(fontName)),
		)
		fontLabel, _, _ := procCreateFontW.Call(
			12, 0, 0, 0, 400, 0, 0, 0, 0, 0, 0, 4, 0,
			uintptr(unsafe.Pointer(fontName)),
		)

		o.mu.Lock()
		o.hwnd = hwnd
		o.fontTitle = fontTitle
		o.fontLabel = fontLabel
		o.stopPing = make(chan struct{})
		o.mu.Unlock()

		go o.pingLoop()

		ready <- nil

		var msg struct {
			Hwnd    uintptr
			Message uint32
			WParam  uintptr
			LParam  uintptr
			Time    uint32
			Pt      struct{ X, Y int32 }
		}

		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if r == 0 || r == uintptr(syscall.InvalidHandle) {
				break
			}
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}()

	return <-ready
}

func (o *PingOverlay) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.stopPing != nil {
		close(o.stopPing)
		o.stopPing = nil
	}
	if o.hwnd != 0 {
		procPostMessageW.Call(o.hwnd, WM_CLOSE, 0, 0)
		if o.fontTitle != 0 {
			procDeleteObject.Call(o.fontTitle)
			o.fontTitle = 0
		}
		if o.fontLabel != 0 {
			procDeleteObject.Call(o.fontLabel)
			o.fontLabel = 0
		}

		var r rect
		if r1, _, _ := procGetWindowRect.Call(o.hwnd, uintptr(unsafe.Pointer(&r))); r1 != 0 {
			o.config.X = r.Left
			o.config.Y = r.Top
		}

		o.hwnd = 0
	}
}

func (o *PingOverlay) pingLoop() {
	o.mu.Lock()
	addrStr := o.config.Address
	stopCh := o.stopPing
	o.mu.Unlock()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	diagSvc := diagnostics.NewService()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			res, err := diagSvc.Ping(addrStr, 1)
			if err != nil || res == nil || len(res.RTTs) == 0 {
				o.updatePing(-1)
			} else {
				o.updatePing(int64(res.RTTs[0]))
			}
		}
	}
}

func (o *PingOverlay) updatePing(ms int64) {
	o.mu.Lock()
	o.currentPing = ms
	hwnd := o.hwnd
	o.mu.Unlock()
	if hwnd != 0 {
		procInvalidateRect.Call(hwnd, 0, 0)
	}
}

func windowProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	o := GetOverlay()

	switch msg {
	case WM_ERASEBKGND:
		return 1
	case WM_NCHITTEST:
		return HTCAPTION
	case WM_PAINT:
		var ps paintstruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			drawOverlayContent(hwnd, hdc, o)
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

func drawOverlayContent(hwnd, hdc uintptr, o *PingOverlay) {
	o.mu.Lock()
	pingVal := o.currentPing
	label := o.config.Label
	if label == "" {
		label = o.config.Address
	}
	fontTitle := o.fontTitle
	fontLabel := o.fontLabel
	bgColor := o.config.BgColorHex
	txtColor := o.config.TextColorHex
	shape := o.config.Shape
	o.mu.Unlock()

	w := int32(100)
	h := int32(44)
	if shape == "circle" {
		w = int32(64)
		h = int32(64)
	}

	memDC, _, _ := procCreateCompatibleDC.Call(hdc)
	memBmp, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(w), uintptr(h))
	oldBmp, _, _ := procSelectObject.Call(memDC, memBmp)

	var bgBGR uint32
	if shape == "circle" {
		bgBGR = 0x00FF00FF
	} else {
		bgBGR = hexToBGR(bgColor, 0x26201C)
	}

	bgBrush, _, _ := procCreateSolidBrush.Call(uintptr(bgBGR))
	bgRect := rect{Left: 0, Top: 0, Right: w, Bottom: h}
	procFillRect.Call(memDC, uintptr(unsafe.Pointer(&bgRect)), bgBrush)
	procDeleteObject.Call(bgBrush)

	var valColor uint32
	if pingVal < 0 {
		valColor = 0x2222EE
	} else if pingVal > 150 {
		valColor = 0x22A5FF
	} else {
		valColor = 0x4AC38B
	}

	if shape == "circle" {
		circleBg := hexToBGR(bgColor, 0x26201C)
		circleBrush, _, _ := procCreateSolidBrush.Call(uintptr(circleBg))
		oldBrush, _, _ := procSelectObject.Call(memDC, circleBrush)
		nullPen, _, _ := procGetStockObject.Call(8)
		oldPen, _, _ := procSelectObject.Call(memDC, nullPen)
		procEllipse.Call(memDC, 0, 0, uintptr(w), uintptr(h))
		procSelectObject.Call(memDC, oldPen)
		procSelectObject.Call(memDC, oldBrush)
		procDeleteObject.Call(circleBrush)
	}

	procSetBkMode.Call(memDC, TRANSPARENT)

	if fontLabel != 0 {
		procSelectObject.Call(memDC, fontLabel)
	}

	lblColor := hexToBGR(txtColor, 0xAAAAAA)
	procSetTextColor.Call(memDC, uintptr(lblColor))

	lblPtr, _ := syscall.UTF16PtrFromString(truncateStr(label, 12))
	var rtLabel rect
	if shape == "circle" {
		rtLabel = rect{Left: 0, Top: 40, Right: w, Bottom: 60}
		procDrawTextW.Call(memDC, uintptr(unsafe.Pointer(lblPtr)), ^uintptr(0), uintptr(unsafe.Pointer(&rtLabel)), 1|4|32|0)
	} else {
		rtLabel = rect{Left: 5, Top: 3, Right: w - 5, Bottom: 20}
		procDrawTextW.Call(memDC, uintptr(unsafe.Pointer(lblPtr)), ^uintptr(0), uintptr(unsafe.Pointer(&rtLabel)), 1|4|32|0)
	}

	if fontTitle != 0 {
		procSelectObject.Call(memDC, fontTitle)
	}

	procSetTextColor.Call(memDC, uintptr(valColor))

	valStr := fmt.Sprintf("%d ms", pingVal)
	if pingVal < 0 {
		valStr = "ERR"
	}

	if shape == "circle" {
		valStr = fmt.Sprintf("%d", pingVal)
		if pingVal < 0 {
			valStr = "ERR"
		}
	}
	valPtr, _ := syscall.UTF16PtrFromString(valStr)

	var rtTitle rect
	if shape == "circle" {
		rtTitle = rect{Left: 0, Top: 15, Right: w, Bottom: 40}
		procDrawTextW.Call(memDC, uintptr(unsafe.Pointer(valPtr)), ^uintptr(0), uintptr(unsafe.Pointer(&rtTitle)), 1|4|32|0)
	} else {
		rtTitle = rect{Left: 5, Top: 18, Right: w - 5, Bottom: h}
		procDrawTextW.Call(memDC, uintptr(unsafe.Pointer(valPtr)), ^uintptr(0), uintptr(unsafe.Pointer(&rtTitle)), 1|4|32|0)
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
		return s[:maxLen-2] + ".."
	}
	return s
}
