//go:build linux

// Package tray shows a status icon on Linux desktops through the
// freedesktop StatusNotifierItem D-Bus protocol (KDE Plasma, GNOME with the
// AppIndicator extension, Cinnamon, XFCE, waybar, ...), with its context
// menu served over com.canonical.dbusmenu.
package tray

import (
	"context"
	"fmt"
	"image"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const (
	itemPath  = dbus.ObjectPath("/StatusNotifierItem")
	itemIface = "org.kde.StatusNotifierItem"

	watcherName  = "org.kde.StatusNotifierWatcher"
	watcherPath  = dbus.ObjectPath("/StatusNotifierWatcher")
	watcherIface = "org.kde.StatusNotifierWatcher"
)

var iconSizes = []int{16, 22, 24, 32, 48, 64}

type MenuItem struct {
	Label     string
	Separator bool
	OnClick   func()
}

type Options struct {
	// ID identifies the application to the tray host; it is never shown.
	ID    string
	Title string
	Icon  image.Image
	Menu  []MenuItem
	// OnActivate runs when the icon itself is clicked.
	OnActivate func()
}

type toolTip struct {
	IconName    string
	IconPixmap  []pixmap
	Title       string
	Description string
}

type Tray struct {
	conn    *dbus.Conn
	props   *prop.Properties
	busName string
	opts    Options

	mu           sync.Mutex
	registered   bool
	closed       bool
	label, guide string
	tip          toolTip
}

// New publishes the icon on the session bus. It succeeds even when no tray
// host is running: the icon registers itself whenever a
// StatusNotifierWatcher appears.
func New(opts Options) (*Tray, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to session bus: %w", err)
	}
	t, err := newWithConn(conn, opts)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return t, nil
}

func newWithConn(conn *dbus.Conn, opts Options) (*Tray, error) {
	opts.ID, opts.Title = dbusString(opts.ID), dbusString(opts.Title)
	opts.Menu = append([]MenuItem(nil), opts.Menu...)
	for i := range opts.Menu {
		opts.Menu[i].Label = dbusString(opts.Menu[i].Label)
	}
	t := &Tray{
		conn:    conn,
		opts:    opts,
		busName: fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid()),
		tip:     toolTip{IconPixmap: []pixmap{}, Title: opts.Title},
	}

	icon := []pixmap{}
	if opts.Icon != nil {
		icon = iconPixmaps(opts.Icon, iconSizes...)
	}
	props, err := prop.Export(conn, itemPath, prop.Map{itemIface: {
		"Category":              {Value: "ApplicationStatus"},
		"Id":                    {Value: opts.ID},
		"Title":                 {Value: opts.Title},
		"Status":                {Value: "Active"},
		"WindowId":              {Value: int32(0)},
		"IconThemePath":         {Value: ""},
		"IconName":              {Value: ""},
		"IconPixmap":            {Value: icon},
		"OverlayIconName":       {Value: ""},
		"OverlayIconPixmap":     {Value: []pixmap{}},
		"AttentionIconName":     {Value: ""},
		"AttentionIconPixmap":   {Value: []pixmap{}},
		"AttentionMovieName":    {Value: ""},
		"ToolTip":               {Value: t.tip},
		"ItemIsMenu":            {Value: false},
		"Menu":                  {Value: menuPath},
		"XAyatanaLabel":         {Value: ""},
		"XAyatanaLabelGuide":    {Value: ""},
		"XAyatanaOrderingIndex": {Value: uint32(0)},
	}})
	if err != nil {
		return nil, err
	}
	t.props = props

	if err := conn.Export(statusNotifierItem{t}, itemPath, itemIface); err != nil {
		return nil, err
	}
	if err := conn.Export(introspect.Introspectable(itemIntrospection), itemPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return nil, err
	}
	if err := exportMenu(conn, opts.Menu); err != nil {
		return nil, err
	}

	// Follow the watcher so the icon comes back when the panel restarts.
	if err := conn.AddMatchSignal(
		dbus.WithMatchSender("org.freedesktop.DBus"),
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchArg(0, watcherName),
	); err != nil {
		return nil, err
	}
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)

	if _, err := conn.RequestName(t.busName, dbus.NameFlagDoNotQueue); err != nil {
		return nil, err
	}
	t.register()
	go t.watch(signals)
	return t, nil
}

func (t *Tray) register() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	call := t.conn.Object(watcherName, watcherPath).CallWithContext(ctx,
		watcherIface+".RegisterStatusNotifierItem", 0, t.busName)
	t.setRegistered(call.Err == nil)
}

func (t *Tray) setRegistered(registered bool) {
	t.mu.Lock()
	t.registered = registered
	t.mu.Unlock()
}

func (t *Tray) watch(signals <-chan *dbus.Signal) {
	for sig := range signals {
		if sig.Name != "org.freedesktop.DBus.NameOwnerChanged" || len(sig.Body) < 3 {
			continue
		}
		if name, _ := sig.Body[0].(string); name != watcherName {
			continue
		}
		if owner, _ := sig.Body[2].(string); owner != "" {
			t.register()
		} else {
			t.setRegistered(false)
		}
	}
}

// Visible reports whether a tray host is currently displaying the icon.
func (t *Tray) Visible() bool {
	t.mu.Lock()
	registered := t.registered && !t.closed
	t.mu.Unlock()
	if !registered {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var hosted dbus.Variant
	err := t.conn.Object(watcherName, watcherPath).CallWithContext(ctx,
		"org.freedesktop.DBus.Properties.Get", 0, watcherIface, "IsStatusNotifierHostRegistered").Store(&hosted)
	if err != nil {
		return false
	}
	ok, _ := hosted.Value().(bool)
	return ok
}

// SetLabel sets the text shown next to the icon by hosts that support
// Ayatana labels (GNOME's AppIndicator extension, Unity, MATE, Budgie).
// guide is the widest expected label; hosts reserve its width so the panel
// does not jitter as the text changes.
func (t *Tray) SetLabel(label, guide string) {
	label, guide = dbusString(label), dbusString(guide)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || (label == t.label && guide == t.guide) {
		return
	}
	t.label, t.guide = label, guide
	t.props.SetMust(itemIface, "XAyatanaLabel", label)
	t.props.SetMust(itemIface, "XAyatanaLabelGuide", guide)
	_ = t.conn.Emit(itemPath, itemIface+".XAyatanaNewLabel", label, guide)
}

func (t *Tray) SetToolTip(title, description string) {
	title, description = dbusString(title), dbusString(description)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || (title == t.tip.Title && description == t.tip.Description) {
		return
	}
	t.tip.Title, t.tip.Description = title, description
	t.props.SetMust(itemIface, "ToolTip", t.tip)
	_ = t.conn.Emit(itemPath, itemIface+".NewToolTip")
}

// dbusString makes s legal on the bus, which only carries valid UTF-8
// without NUL bytes; process names come from file names and may be neither.
func dbusString(s string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(s, "�"), "\x00", "")
}

// Close removes the icon and disconnects from the session bus.
func (t *Tray) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.closed = true
	t.registered = false
	_, _ = t.conn.ReleaseName(t.busName)
	_ = t.conn.Close()
}

type statusNotifierItem struct {
	t *Tray
}

func (s statusNotifierItem) Activate(x, y int32) *dbus.Error {
	if fn := s.t.opts.OnActivate; fn != nil {
		go fn()
	}
	return nil
}

func (s statusNotifierItem) SecondaryActivate(x, y int32) *dbus.Error {
	return nil
}

// ContextMenu is only used by hosts that ignore the Menu property; the menu
// cannot be drawn without a toolkit, so there is nothing to do.
func (s statusNotifierItem) ContextMenu(x, y int32) *dbus.Error {
	return nil
}

func (s statusNotifierItem) Scroll(delta int32, orientation string) *dbus.Error {
	return nil
}

const itemIntrospection = `<node>
	<interface name="org.kde.StatusNotifierItem">
		<property name="Category" type="s" access="read"/>
		<property name="Id" type="s" access="read"/>
		<property name="Title" type="s" access="read"/>
		<property name="Status" type="s" access="read"/>
		<property name="WindowId" type="i" access="read"/>
		<property name="IconThemePath" type="s" access="read"/>
		<property name="IconName" type="s" access="read"/>
		<property name="IconPixmap" type="a(iiay)" access="read"/>
		<property name="OverlayIconName" type="s" access="read"/>
		<property name="OverlayIconPixmap" type="a(iiay)" access="read"/>
		<property name="AttentionIconName" type="s" access="read"/>
		<property name="AttentionIconPixmap" type="a(iiay)" access="read"/>
		<property name="AttentionMovieName" type="s" access="read"/>
		<property name="ToolTip" type="(sa(iiay)ss)" access="read"/>
		<property name="ItemIsMenu" type="b" access="read"/>
		<property name="Menu" type="o" access="read"/>
		<property name="XAyatanaLabel" type="s" access="read"/>
		<property name="XAyatanaLabelGuide" type="s" access="read"/>
		<property name="XAyatanaOrderingIndex" type="u" access="read"/>
		<method name="ContextMenu">
			<arg name="x" type="i" direction="in"/>
			<arg name="y" type="i" direction="in"/>
		</method>
		<method name="Activate">
			<arg name="x" type="i" direction="in"/>
			<arg name="y" type="i" direction="in"/>
		</method>
		<method name="SecondaryActivate">
			<arg name="x" type="i" direction="in"/>
			<arg name="y" type="i" direction="in"/>
		</method>
		<method name="Scroll">
			<arg name="delta" type="i" direction="in"/>
			<arg name="orientation" type="s" direction="in"/>
		</method>
		<signal name="NewTitle"/>
		<signal name="NewIcon"/>
		<signal name="NewAttentionIcon"/>
		<signal name="NewOverlayIcon"/>
		<signal name="NewToolTip"/>
		<signal name="NewStatus">
			<arg name="status" type="s"/>
		</signal>
		<signal name="XAyatanaNewLabel">
			<arg name="label" type="s"/>
			<arg name="guide" type="s"/>
		</signal>
	</interface>` + prop.IntrospectDataString + introspect.IntrospectDataString + `</node>`
