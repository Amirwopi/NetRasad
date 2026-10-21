//go:build linux

package netinfo

import (
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// hexIPv4 renders an address the way /proc/net/route prints it on this host.
func hexIPv4(s string) string {
	return fmt.Sprintf("%08X", binary.NativeEndian.Uint32(net.ParseIP(s).To4()))
}

func TestParseDefaultRoutes(t *testing.T) {
	table := "Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT\n" +
		fmt.Sprintf("wlan0\t00000000\t%s\t0003\t0\t0\t600\t00000000\t0\t0\t0\n", hexIPv4("192.168.1.1")) +
		fmt.Sprintf("eth0\t00000000\t%s\t0003\t0\t9999\t100\t00000000\t0\t0\t0\n", hexIPv4("10.0.0.1")) +
		// Point-to-point default route without a gateway (e.g. a VPN tunnel).
		"wg0\t00000000\t00000000\t0001\t0\t0\t50\t00000000\t0\t0\t0\n" +
		// Not a default route.
		fmt.Sprintf("eth0\t%s\t00000000\t0001\t0\t0\t100\t%s\t0\t0\t0\n", hexIPv4("10.0.0.0"), hexIPv4("255.255.255.0")) +
		// Default route that is not up.
		fmt.Sprintf("eth1\t00000000\t%s\t0002\t0\t0\t10\t00000000\t0\t0\t0\n", hexIPv4("172.16.0.1"))

	routes, err := parseDefaultRoutes(strings.NewReader(table))
	if err != nil {
		t.Fatal(err)
	}
	want := []Route{
		{Interface: "eth0", Gateway: "10.0.0.1", Metric: 100},
		{Interface: "wlan0", Gateway: "192.168.1.1", Metric: 600},
	}
	if len(routes) != len(want) {
		t.Fatalf("routes = %+v, want %+v", routes, want)
	}
	for i := range want {
		if routes[i] != want[i] {
			t.Errorf("routes[%d] = %+v, want %+v", i, routes[i], want[i])
		}
	}
}

func TestParseHexIPv4(t *testing.T) {
	got, err := parseHexIPv4(hexIPv4("192.168.10.254"))
	if err != nil || got != "192.168.10.254" {
		t.Fatalf("parseHexIPv4 = %q, %v", got, err)
	}
	if _, err := parseHexIPv4("zz"); err == nil {
		t.Fatal("expected error for invalid hex")
	}
}

func TestFormatIPsPutsIPv4First(t *testing.T) {
	got := formatIPs([]net.IP{
		net.ParseIP("fe80::1"),
		net.ParseIP("192.168.1.20"),
		net.ParseIP("2001:db8::5"),
		net.ParseIP("10.8.0.2"),
	})
	want := []string{"192.168.1.20", "10.8.0.2", "fe80::1", "2001:db8::5"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("formatIPs = %v, want %v", got, want)
	}
	if got := formatIPs(nil); got == nil || len(got) != 0 {
		t.Fatalf("formatIPs(nil) = %#v, want empty non-nil slice", got)
	}
}

func TestAddressesByIndexMatchesStdlib(t *testing.T) {
	got, err := addressesByIndex()
	if err != nil {
		t.Fatal(err)
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, ifi := range ifaces {
		addrs, err := ifi.Addrs()
		if err != nil {
			t.Fatal(err)
		}
		var want []string
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				want = append(want, ipn.IP.String())
			}
		}
		var have []string
		for _, ip := range got[ifi.Index] {
			have = append(have, ip.String())
		}
		sort.Strings(want)
		sort.Strings(have)
		if strings.Join(have, ",") != strings.Join(want, ",") {
			t.Errorf("%s: addresses = %v, stdlib says %v", ifi.Name, have, want)
		}
	}
}

func genlMessage(attrs ...[]byte) []byte {
	msg := []byte{unix.NL80211_CMD_NEW_INTERFACE, 1, 0, 0}
	for _, a := range attrs {
		msg = append(msg, a...)
	}
	return msg
}

func u32(v uint32) []byte {
	b := make([]byte, 4)
	binary.NativeEndian.PutUint32(b, v)
	return b
}

func TestSSIDsFromInterfaceDump(t *testing.T) {
	msgs := [][]byte{
		genlMessage(
			encodeAttr(unix.NL80211_ATTR_IFINDEX, u32(3)),
			encodeAttr(unix.NL80211_ATTR_IFNAME, []byte("wlan0\x00")),
			encodeAttr(unix.NL80211_ATTR_SSID, []byte("Home WiFi")),
		),
		// Disconnected interface: no SSID attribute.
		genlMessage(
			encodeAttr(unix.NL80211_ATTR_IFINDEX, u32(4)),
			encodeAttr(unix.NL80211_ATTR_IFNAME, []byte("wlan1\x00")),
		),
		// Hidden network reporting a zeroed SSID element.
		genlMessage(
			encodeAttr(unix.NL80211_ATTR_IFNAME, []byte("wlan2\x00")),
			encodeAttr(unix.NL80211_ATTR_SSID, make([]byte, 6)),
		),
		// Odd-length SSID followed by another attribute exercises padding.
		genlMessage(
			encodeAttr(unix.NL80211_ATTR_SSID, []byte("abc")),
			encodeAttr(unix.NL80211_ATTR_IFNAME, []byte("wlp2s0\x00")),
		),
		{0x01},
	}
	got := ssidsFromInterfaceDump(msgs)
	want := map[string]string{"wlan0": "Home WiFi", "wlp2s0": "abc"}
	if len(got) != len(want) {
		t.Fatalf("ssids = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("ssids[%q] = %q, want %q", k, got[k], v)
		}
	}
}

func TestParseAttrsStopsOnMalformedInput(t *testing.T) {
	good := encodeAttr(unix.NL80211_ATTR_IFINDEX, u32(7))
	bad := []byte{0xff, 0x00, 0x04, 0x00} // claims 255 bytes
	attrs := parseAttrs(append(append([]byte{}, good...), bad...))
	if v := attrs[unix.NL80211_ATTR_IFINDEX]; len(v) != 4 || binary.NativeEndian.Uint32(v) != 7 {
		t.Fatalf("IFINDEX attr = %v", v)
	}
	if len(attrs) != 1 {
		t.Fatalf("parsed %d attrs, want 1", len(attrs))
	}
}

func TestGetAdapterInfoLoopback(t *testing.T) {
	info := GetAdapterInfo()
	lo, ok := info["lo"]
	if !ok {
		t.Skip("no loopback interface in this environment")
	}
	found := false
	for _, ip := range lo.IPAddresses {
		if ip == "127.0.0.1" {
			found = true
		}
	}
	if !found {
		t.Errorf("lo addresses = %v, want 127.0.0.1 among them", lo.IPAddresses)
	}
}

func TestWifiSSIDsNeverFails(t *testing.T) {
	// Exercises the real nl80211 round trip; machines without Wi-Fi return an empty map.
	if got := wifiSSIDs(); got == nil {
		t.Fatal("wifiSSIDs returned nil map")
	}
}

func TestGenetlinkRoundTrip(t *testing.T) {
	c, err := dialGenetlink()
	if err != nil {
		t.Skipf("generic netlink unavailable: %v", err)
	}
	defer c.close()

	// Single-reply request: the controller resolves its own family.
	id, err := c.familyID("nlctrl")
	if err != nil {
		t.Fatal(err)
	}
	if id != unix.GENL_ID_CTRL {
		t.Fatalf("nlctrl family id = %d, want %d", id, unix.GENL_ID_CTRL)
	}

	// Multi-message dump terminated by NLMSG_DONE.
	msgs, err := c.request(unix.GENL_ID_CTRL, unix.CTRL_CMD_GETFAMILY, unix.NLM_F_DUMP, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, m := range msgs {
		name := parseAttrs(m[unix.GENL_HDRLEN:])[unix.CTRL_ATTR_FAMILY_NAME]
		names[strings.TrimRight(string(name), "\x00")] = true
	}
	if !names["nlctrl"] {
		t.Fatalf("family dump did not include nlctrl: %v", names)
	}

	if _, err := c.familyID("netrasad-no-such-family"); err == nil {
		t.Fatal("expected an error for an unknown family")
	}
}
