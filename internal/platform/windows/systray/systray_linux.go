//go:build linux

package systray

import (
	"bytes"
	"image"
	"image/png"
	"sync"

	"github.com/netrasad/netrasad/internal/platform/linux/tray"
)

// SysTray shows NetRasad through the desktop's StatusNotifierItem host. Its
// label and tooltip carry the live speed that Windows draws on the taskbar.
type SysTray struct {
	mu       sync.Mutex
	tray     *tray.Tray
	icon     []byte
	label    string
	guide    string
	tipTitle string
	tipBody  string
}

var globalTray = &SysTray{tipTitle: "NetRasad"}

func GetSysTray() *SysTray {
	return globalTray
}

// SetIcon sets the PNG shown in the tray; it must be called before Start.
func (t *SysTray) SetIcon(pngData []byte) {
	t.mu.Lock()
	t.icon = pngData
	t.mu.Unlock()
}

func (t *SysTray) Start(onOpen, onToggleBand, onExit func()) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.tray != nil {
		return nil
	}

	var icon image.Image
	if len(t.icon) > 0 {
		icon, _ = png.Decode(bytes.NewReader(t.icon))
	}
	tr, err := tray.New(tray.Options{
		ID:    "netrasad",
		Title: "NetRasad",
		Icon:  icon,
		Menu: []tray.MenuItem{
			{Label: "Open NetRasad", OnClick: onOpen},
			{Label: "Toggle Speed Label", OnClick: onToggleBand},
			{Separator: true},
			{Label: "Exit NetRasad", OnClick: onExit},
		},
		OnActivate: onOpen,
	})
	if err != nil {
		return err
	}
	t.tray = tr
	tr.SetLabel(t.label, t.guide)
	tr.SetToolTip(t.tipTitle, t.tipBody)
	return nil
}

func (t *SysTray) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.tray != nil {
		t.tray.Close()
		t.tray = nil
	}
}

// Visible reports whether a tray host is showing the icon, i.e. whether a
// hidden window could be brought back from it.
func (t *SysTray) Visible() bool {
	t.mu.Lock()
	tr := t.tray
	t.mu.Unlock()
	return tr != nil && tr.Visible()
}

// SetStatus updates the text next to the icon and its tooltip.
func (t *SysTray) SetStatus(label, guide, tipTitle, tipBody string) {
	t.mu.Lock()
	t.label, t.guide, t.tipTitle, t.tipBody = label, guide, tipTitle, tipBody
	tr := t.tray
	t.mu.Unlock()
	if tr != nil {
		tr.SetLabel(label, guide)
		tr.SetToolTip(tipTitle, tipBody)
	}
}
