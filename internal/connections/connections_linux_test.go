//go:build linux

package connections

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// procHex renders an address the way /proc/net/{tcp,udp}{,6} prints it on this host.
func procHex(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil && !strings.Contains(ip.String(), ":") {
		ip = v4
	} else {
		ip = ip.To16()
	}
	var sb strings.Builder
	for i := 0; i < len(ip); i += 4 {
		fmt.Fprintf(&sb, "%08X", binary.NativeEndian.Uint32(ip[i:i+4]))
	}
	return sb.String()
}

func TestParseAddrPort(t *testing.T) {
	tests := []struct {
		ip   string
		isV6 bool
		want string
	}{
		{"127.0.0.1", false, "127.0.0.1"},
		{"192.168.1.20", false, "192.168.1.20"},
		{"::1", true, "::1"},
		{"2001:db8::8a2e:370:7334", true, "2001:db8::8a2e:370:7334"},
		{"fe80::1ff:fe23:4567:890a", true, "fe80::1ff:fe23:4567:890a"},
		{"::", true, "::"},
	}
	for _, tt := range tests {
		raw := procHex(net.ParseIP(tt.ip))
		if tt.isV6 {
			raw = procHex(net.ParseIP(tt.ip).To16())
		}
		got, port, err := parseAddrPort(raw+":01BB", tt.isV6)
		if err != nil {
			t.Fatalf("parseAddrPort(%s): %v", tt.ip, err)
		}
		if got != tt.want || port != 443 {
			t.Errorf("parseAddrPort(%s) = %s:%d, want %s:443", tt.ip, got, port, tt.want)
		}
	}

	// IPv4-mapped addresses on dual-stack sockets render in dotted form, as on Windows.
	mapped := make([]byte, 16)
	copy(mapped[10:], []byte{0xff, 0xff, 10, 0, 0, 7})
	var sb strings.Builder
	for i := 0; i < 16; i += 4 {
		fmt.Fprintf(&sb, "%08X", binary.NativeEndian.Uint32(mapped[i:i+4]))
	}
	if got, _, err := parseAddrPort(sb.String()+":0050", true); err != nil || got != "10.0.0.7" {
		t.Errorf("mapped address = %q, %v; want 10.0.0.7", got, err)
	}

	for _, bad := range []string{"0100007F", "0100007F:ZZZZ", "7F:0050", "XX00007F:0050"} {
		if _, _, err := parseAddrPort(bad, false); err == nil {
			t.Errorf("parseAddrPort(%q) succeeded, want error", bad)
		}
	}
}

func TestParseProcNet(t *testing.T) {
	table := "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n" +
		fmt.Sprintf("   0: %s:0035 00000000:0000 0A 00000000:00000000 00:00000000 00000000   991        0 21456 1 0000000000000000 100 0 0 10 0\n", procHex(net.ParseIP("127.0.0.53"))) +
		fmt.Sprintf("   1: %s:D2F4 %s:01BB 01 00000000:00000000 02:000A7B2C 00000000  1000        0 98765 2 0000000000000000 20 4 30 10 -1\n", procHex(net.ParseIP("192.168.1.101")), procHex(net.ParseIP("140.82.112.4"))) +
		"   2: garbage\n"

	entries, err := parseProcNet(strings.NewReader(table), TCP, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	listen := entries[0]
	if listen.conn.LocalAddress != "127.0.0.53" || listen.conn.LocalPort != 53 || listen.conn.State != StateListen || listen.inode != 21456 {
		t.Errorf("listen entry = %+v", listen)
	}
	est := entries[1]
	if est.conn.RemoteAddress != "140.82.112.4" || est.conn.RemotePort != 443 || est.conn.LocalPort != 0xD2F4 || est.conn.State != StateEstablished {
		t.Errorf("established entry = %+v", est)
	}

	udp, err := parseProcNet(strings.NewReader(table), UDP, false)
	if err != nil {
		t.Fatal(err)
	}
	if udp[0].conn.State != StateNA {
		t.Errorf("udp state = %s, want %s", udp[0].conn.State, StateNA)
	}
}

func TestOwnerRescanPolicy(t *testing.T) {
	o := &socketOwners{}
	entries := []socketEntry{{inode: 10}, {inode: 0}}
	if !o.needsRescan(entries) {
		t.Fatal("empty cache must scan")
	}

	o.pids = map[uint64]int{10: 42}
	o.seen = map[uint64]bool{10: true, 11: true}
	o.lastScan = time.Now().Add(-time.Hour)
	if o.needsRescan(append(entries, socketEntry{inode: 11})) {
		t.Error("only known sockets (and an ownerless inode 0) must not trigger a rescan")
	}
	if !o.needsRescan(append(entries, socketEntry{inode: 12})) {
		t.Error("an unseen socket must trigger a rescan")
	}

	o.lastScan = time.Now()
	if o.needsRescan(append(entries, socketEntry{inode: 12})) {
		t.Error("rescans must be rate limited")
	}
}

func TestGetConnectionsFindsOwnSockets(t *testing.T) {
	owners = &socketOwners{}

	ln4, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln4.Close()
	targets := map[int]string{ln4.Addr().(*net.TCPAddr).Port: "127.0.0.1"}

	if ln6, err := net.Listen("tcp6", "[::1]:0"); err == nil {
		defer ln6.Close()
		targets[ln6.Addr().(*net.TCPAddr).Port] = "::1"
	} else {
		t.Logf("IPv6 loopback unavailable, checking IPv4 only: %v", err)
	}

	conns, err := NewService().GetConnections()
	if err != nil {
		t.Fatal(err)
	}
	exe, _ := os.Executable()
	for port, addr := range targets {
		var found *Connection
		for i := range conns {
			if conns[i].LocalPort == port && conns[i].State == StateListen {
				found = &conns[i]
			}
		}
		if found == nil {
			t.Errorf("listener on %s:%d not reported", addr, port)
			continue
		}
		if found.LocalAddress != addr {
			t.Errorf("listener address = %s, want %s", found.LocalAddress, addr)
		}
		if found.PID != os.Getpid() {
			t.Errorf("listener pid = %d, want %d", found.PID, os.Getpid())
		}
		if found.ProcessName != filepath.Base(exe) {
			t.Errorf("listener process = %q, want %q", found.ProcessName, filepath.Base(exe))
		}
	}
}
