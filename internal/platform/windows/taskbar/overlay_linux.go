//go:build linux

package taskbar

import (
	"fmt"
	"strings"
	"sync"

	"github.com/netrasad/netrasad/internal/platform/windows/systray"
)

// There is no taskbar to draw on under Linux, so the speed band becomes the
// label next to the tray icon (shown by hosts that support Ayatana labels,
// such as GNOME with the AppIndicator extension) and the icon's tooltip,
// which also lists the busiest applications like the Windows band does.

// labelGuide is the widest label formatRateMbit can produce; hosts reserve
// its width so the panel does not shift as the numbers change.
const labelGuide = "▼ 1023.9 KB/s ▲ 1023.9 KB/s"

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
	Monitor       int32  `json:"monitor"`
}

type TaskbarOverlay struct {
	mu           sync.Mutex
	enabled      bool
	config       TaskbarOverlayConfig
	downRate     float64
	upRate       float64
	topProcesses []TopProcessEntry
}

var globalOverlay = &TaskbarOverlay{}

func GetOverlay() *TaskbarOverlay {
	return globalOverlay
}

func (o *TaskbarOverlay) Start() error {
	o.mu.Lock()
	o.enabled = true
	o.mu.Unlock()
	o.publish()
	return nil
}

func (o *TaskbarOverlay) Stop() {
	o.mu.Lock()
	o.enabled = false
	o.mu.Unlock()
	o.publish()
}

func (o *TaskbarOverlay) SetEnabled(enabled bool) {
	o.mu.Lock()
	o.enabled = enabled
	o.mu.Unlock()
	o.publish()
}

func (o *TaskbarOverlay) IsEnabled() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.enabled
}

// SetConfig keeps the settings so they round-trip through the UI; colours,
// size and placement have no equivalent in a panel label.
func (o *TaskbarOverlay) SetConfig(cfg TaskbarOverlayConfig) {
	o.mu.Lock()
	o.config = cfg
	o.mu.Unlock()
}

func (o *TaskbarOverlay) GetConfig() TaskbarOverlayConfig {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.config
}

func (o *TaskbarOverlay) UpdateTopProcesses(list []TopProcessEntry) {
	o.mu.Lock()
	if len(list) > 3 {
		list = list[:3]
	}
	o.topProcesses = append([]TopProcessEntry(nil), list...)
	o.mu.Unlock()
}

func (o *TaskbarOverlay) UpdateRates(downBps, upBps float64) {
	o.mu.Lock()
	o.downRate = downBps
	o.upRate = upBps
	o.mu.Unlock()
	o.publish()
}

func (o *TaskbarOverlay) publish() {
	label, guide, tip := o.status()
	systray.GetSysTray().SetStatus(label, guide, "NetRasad", tip)
}

func (o *TaskbarOverlay) status() (label, guide, tip string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.enabled {
		label = fmt.Sprintf("▼ %s ▲ %s", formatRateMbit(o.downRate), formatRateMbit(o.upRate))
		guide = labelGuide
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "▼ %s   ▲ %s", formatRateMbit(o.downRate), formatRateMbit(o.upRate))
	if len(o.topProcesses) > 0 {
		sb.WriteString("\n\nApplications")
		for _, p := range o.topProcesses {
			fmt.Fprintf(&sb, "\n%s   ▼ %s ▲ %s", truncateStr(p.Name, 25), formatRateMbit(p.DownloadBps), formatRateMbit(p.UploadBps))
		}
	}
	return label, guide, sb.String()
}

// truncateStr counts runes so multi-byte names are never cut mid-character.
func truncateStr(s string, maxLen int) string {
	if r := []rune(s); len(r) > maxLen {
		return string(r[:maxLen-1]) + "…"
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
