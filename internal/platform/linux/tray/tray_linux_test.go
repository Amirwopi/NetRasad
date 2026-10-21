//go:build linux

package tray

import (
	"bufio"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

func TestOpaqueSquareCentresContent(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := 4; y < 6; y++ {
		for x := 3; x < 7; x++ {
			img.Set(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	if got, want := opaqueSquare(img), image.Rect(3, 3, 7, 7); got != want {
		t.Fatalf("opaqueSquare = %v, want %v", got, want)
	}

	empty := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	if got := opaqueSquare(empty); got != empty.Bounds() {
		t.Fatalf("fully transparent image: got %v, want its bounds", got)
	}
}

func TestScaleSquareToPixmap(t *testing.T) {
	solid := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := range solid.Pix {
		solid.Pix[i] = []byte{0x10, 0x20, 0x30, 0xff}[i%4]
	}
	pm := toPixmap(scaleSquare(solid, solid.Bounds(), 2))
	if pm.Width != 2 || pm.Height != 2 || len(pm.Data) != 2*2*4 {
		t.Fatalf("pixmap = %dx%d with %d bytes", pm.Width, pm.Height, len(pm.Data))
	}
	if got := pm.Data[:4]; string(got) != string([]byte{0xff, 0x10, 0x20, 0x30}) {
		t.Fatalf("first pixel = % x, want ARGB ff 10 20 30", got)
	}

	// Averaging an opaque red pixel with a transparent black one must keep the
	// colour red at half alpha instead of darkening it.
	edge := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	edge.Set(0, 0, color.NRGBA{R: 255, A: 255})
	edge.Set(0, 1, color.NRGBA{R: 255, A: 255})
	pm = toPixmap(scaleSquare(edge, edge.Bounds(), 1))
	if a, r, g, b := pm.Data[0], pm.Data[1], pm.Data[2], pm.Data[3]; a < 0x7e || a > 0x80 || r != 0xff || g != 0 || b != 0 {
		t.Fatalf("edge pixel = % x, want ~7f ff 00 00", pm.Data)
	}
}

func TestIconPixmapsFromAppIcon(t *testing.T) {
	f, err := os.Open("../../../../build/appicon.png")
	if err != nil {
		t.Skipf("app icon not available: %v", err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	pms := iconPixmaps(img, iconSizes...)
	if len(pms) != len(iconSizes) {
		t.Fatalf("got %d pixmaps, want %d", len(pms), len(iconSizes))
	}
	for i, pm := range pms {
		if int(pm.Width) != iconSizes[i] || len(pm.Data) != iconSizes[i]*iconSizes[i]*4 {
			t.Errorf("pixmap %d is %dx%d with %d bytes", i, pm.Width, pm.Height, len(pm.Data))
		}
	}
	// The source has wide transparent margins; after cropping, a point one
	// eighth in from the left edge must land on the artwork.
	big := pms[len(pms)-1]
	if alpha := big.Data[(32*64+8)*4]; alpha == 0 {
		t.Error("icon margin was not cropped: left-middle pixel is transparent")
	}
}

func TestMenuLayoutAndProperties(t *testing.T) {
	m := &dbusMenu{items: []MenuItem{{Label: "Open"}, {Separator: true}, {Label: "Exit"}}}

	rev, root, derr := m.GetLayout(0, -1, nil)
	if derr != nil {
		t.Fatal(derr)
	}
	if rev != menuRevision || root.ID != 0 || len(root.Children) != 3 {
		t.Fatalf("root layout = rev %d %+v", rev, root)
	}
	sep := root.Children[1].Value().(menuLayout)
	if sep.ID != 2 || sep.Properties["type"].Value() != "separator" {
		t.Errorf("separator layout = %+v", sep)
	}
	if exit := root.Children[2].Value().(menuLayout); exit.Properties["label"].Value() != "Exit" {
		t.Errorf("exit layout = %+v", exit)
	}

	if _, shallow, _ := m.GetLayout(0, 0, nil); len(shallow.Children) != 0 {
		t.Errorf("depth 0 must not include children, got %d", len(shallow.Children))
	}
	if _, filtered, _ := m.GetLayout(1, -1, []string{"label"}); len(filtered.Properties) != 1 {
		t.Errorf("filtered properties = %v", filtered.Properties)
	}
	if _, _, derr := m.GetLayout(9, -1, nil); derr == nil {
		t.Error("unknown parent must fail")
	}

	all, _ := m.GetGroupProperties(nil, nil)
	if len(all) != 3 {
		t.Errorf("GetGroupProperties(all) returned %d items", len(all))
	}
	if v, derr := m.GetProperty(3, "label"); derr != nil || v.Value() != "Exit" {
		t.Errorf("GetProperty(3, label) = %v, %v", v, derr)
	}
	if _, derr := m.GetProperty(3, "icon-name"); derr == nil {
		t.Error("missing property must fail")
	}
	if errs, _ := m.EventGroup([]menuEvent{{ID: 1, EventID: "hovered"}, {ID: 42, EventID: "clicked"}}); len(errs) != 1 || errs[0] != 42 {
		t.Errorf("EventGroup idErrors = %v, want [42]", errs)
	}
}

// startBus runs a private session bus so the protocol can be exercised
// without touching the desktop the tests run on.
func startBus(t *testing.T) string {
	t.Helper()
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon not installed")
	}
	cmd := exec.Command(daemon, "--session", "--nofork", "--print-address")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start dbus-daemon: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})
	addr, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("read bus address: %v", err)
	}
	return strings.TrimSpace(addr)
}

type fakeWatcher struct {
	registrations chan string
}

func (w *fakeWatcher) RegisterStatusNotifierItem(service string) *dbus.Error {
	w.registrations <- service
	return nil
}

func connect(t *testing.T, addr string) *dbus.Conn {
	t.Helper()
	conn, err := dbus.Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func expectRegistration(t *testing.T, w *fakeWatcher, want string) {
	t.Helper()
	select {
	case got := <-w.registrations:
		if got != want {
			t.Fatalf("registered %q, want %q", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("item never registered with the watcher")
	}
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestTrayOverDBus(t *testing.T) {
	addr := startBus(t)
	host := connect(t, addr)

	watcher := &fakeWatcher{registrations: make(chan string, 4)}
	if err := host.Export(watcher, watcherPath, watcherIface); err != nil {
		t.Fatal(err)
	}
	hostProps, err := prop.Export(host, watcherPath, prop.Map{watcherIface: {
		"IsStatusNotifierHostRegistered": {Value: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := host.RequestName(watcherName, dbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}

	icon := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for i := range icon.Pix {
		icon.Pix[i] = 0xff
	}
	var mu sync.Mutex
	var clicked []string
	record := func(what string) func() {
		return func() {
			mu.Lock()
			clicked = append(clicked, what)
			mu.Unlock()
		}
	}
	hasClicked := func(what string) func() bool {
		return func() bool {
			mu.Lock()
			defer mu.Unlock()
			for _, c := range clicked {
				if c == what {
					return true
				}
			}
			return false
		}
	}

	itemConn, err := dbus.Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	tr, err := newWithConn(itemConn, Options{
		ID:    "netrasad-test",
		Title: "NetRasad",
		Icon:  icon,
		Menu: []MenuItem{
			{Label: "Open NetRasad", OnClick: record("open")},
			{Separator: true},
			{Label: "Exit NetRasad", OnClick: record("exit")},
		},
		OnActivate: record("activate"),
	})
	if err != nil {
		itemConn.Close()
		t.Fatal(err)
	}
	defer tr.Close()

	expectRegistration(t, watcher, tr.busName)
	if !tr.Visible() {
		t.Fatal("Visible() = false with a registered host")
	}

	item := host.Object(tr.busName, itemPath)
	call := item.Call("org.freedesktop.DBus.Properties.GetAll", 0, itemIface)
	if call.Err != nil {
		t.Fatal(call.Err)
	}
	// Read the body directly: Store rebuilds variants and drops their wire signatures.
	props := call.Body[0].(map[string]dbus.Variant)
	if props["Id"].Value() != "netrasad-test" || props["Title"].Value() != "NetRasad" || props["Status"].Value() != "Active" {
		t.Errorf("identity properties = %v", props)
	}
	if props["ItemIsMenu"].Value() != false || props["Menu"].Value() != menuPath {
		t.Errorf("menu properties = %v / %v", props["ItemIsMenu"], props["Menu"])
	}
	if sig := props["IconPixmap"].Signature().String(); sig != "a(iiay)" {
		t.Errorf("IconPixmap signature = %s", sig)
	}
	if sig := props["ToolTip"].Signature().String(); sig != "(sa(iiay)ss)" {
		t.Errorf("ToolTip signature = %s", sig)
	}
	if pms := props["IconPixmap"].Value().([][]interface{}); len(pms) != len(iconSizes) {
		t.Errorf("got %d icon pixmaps, want %d", len(pms), len(iconSizes))
	}

	var xmlData string
	if err := item.Call("org.freedesktop.DBus.Introspectable.Introspect", 0).Store(&xmlData); err != nil {
		t.Fatal(err)
	}
	var node introspect.Node
	if err := xml.Unmarshal([]byte(xmlData), &node); err != nil {
		t.Fatalf("introspection XML is invalid: %v", err)
	}
	ifaces := map[string]bool{}
	for _, i := range node.Interfaces {
		ifaces[i.Name] = true
	}
	for _, want := range []string{itemIface, "org.freedesktop.DBus.Properties", "org.freedesktop.DBus.Introspectable"} {
		if !ifaces[want] {
			t.Errorf("introspection lacks %s", want)
		}
	}

	menu := host.Object(tr.busName, menuPath)
	var revision uint32
	var layout []interface{}
	if err := menu.Call(menuIface+".GetLayout", 0, int32(0), int32(-1), []string{}).Store(&revision, &layout); err != nil {
		t.Fatal(err)
	}
	children := layout[2].([]dbus.Variant)
	if len(children) != 3 {
		t.Fatalf("menu has %d entries, want 3", len(children))
	}
	first := children[0].Value().([]interface{})
	if label := first[1].(map[string]dbus.Variant)["label"].Value(); label != "Open NetRasad" {
		t.Errorf("first menu label = %v", label)
	}
	var version uint32
	if err := menu.Call("org.freedesktop.DBus.Properties.Get", 0, menuIface, "Version").Store(&version); err != nil || version != 3 {
		t.Errorf("dbusmenu Version = %d, %v", version, err)
	}

	if err := menu.Call(menuIface+".Event", 0, int32(3), "clicked", dbus.MakeVariant(int32(0)), uint32(0)).Err; err != nil {
		t.Fatal(err)
	}
	eventually(t, "exit click", hasClicked("exit"))
	if err := item.Call(itemIface+".Activate", 0, int32(0), int32(0)).Err; err != nil {
		t.Fatal(err)
	}
	eventually(t, "activation", hasClicked("activate"))

	if err := host.AddMatchSignal(dbus.WithMatchInterface(itemIface)); err != nil {
		t.Fatal(err)
	}
	signals := make(chan *dbus.Signal, 16)
	host.Signal(signals)
	waitSignal := func(member string) *dbus.Signal {
		t.Helper()
		timeout := time.After(5 * time.Second)
		for {
			select {
			case s := <-signals:
				if s.Name == itemIface+"."+member {
					return s
				}
			case <-timeout:
				t.Fatalf("no %s signal", member)
			}
		}
	}

	tr.SetLabel("▼ 1.0 KB/s ▲ 2 B/s", "▼ 000.0 KB/s ▲ 000.0 KB/s")
	if s := waitSignal("XAyatanaNewLabel"); len(s.Body) != 2 || s.Body[0] != "▼ 1.0 KB/s ▲ 2 B/s" {
		t.Errorf("XAyatanaNewLabel body = %v", s.Body)
	}
	var label string
	if err := item.Call("org.freedesktop.DBus.Properties.Get", 0, itemIface, "XAyatanaLabel").Store(&label); err != nil || label != "▼ 1.0 KB/s ▲ 2 B/s" {
		t.Errorf("XAyatanaLabel = %q, %v", label, err)
	}

	// Strings that are not valid UTF-8 would otherwise break the connection.
	tr.SetLabel("bad\xffname\x00", "")
	if s := waitSignal("XAyatanaNewLabel"); len(s.Body) != 2 || s.Body[0] != "bad\uFFFDname" {
		t.Errorf("sanitised label signal = %v", s.Body)
	}

	tr.SetToolTip("NetRasad", "▼ 1.0 KB/s")
	waitSignal("NewToolTip")
	var tip []interface{}
	if err := item.Call("org.freedesktop.DBus.Properties.Get", 0, itemIface, "ToolTip").Store(&tip); err != nil {
		t.Fatal(err)
	}
	if tip[2] != "NetRasad" || tip[3] != "▼ 1.0 KB/s" {
		t.Errorf("ToolTip = %v", tip)
	}

	hostProps.SetMust(watcherIface, "IsStatusNotifierHostRegistered", false)
	if tr.Visible() {
		t.Error("Visible() = true while no host is registered")
	}
	hostProps.SetMust(watcherIface, "IsStatusNotifierHostRegistered", true)

	// A restarted panel brings a new watcher; the icon must register again.
	if _, err := host.ReleaseName(watcherName); err != nil {
		t.Fatal(err)
	}
	eventually(t, "unregistration", func() bool { return !tr.Visible() })
	if _, err := host.RequestName(watcherName, dbus.NameFlagDoNotQueue); err != nil {
		t.Fatal(err)
	}
	expectRegistration(t, watcher, tr.busName)
	eventually(t, "re-registration", tr.Visible)

	tr.Close()
	if tr.Visible() {
		t.Error("Visible() = true after Close")
	}
	var owned bool
	if err := host.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, tr.busName).Store(&owned); err != nil || owned {
		t.Errorf("bus name still owned after Close (%v)", err)
	}
}
