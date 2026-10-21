//go:build linux

package traffic

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/netrasad/netrasad/internal/domain"
)

func TestMapLinuxIfType(t *testing.T) {
	tests := []struct {
		name     string
		arphrd   int
		wireless bool
		want     domain.InterfaceType
	}{
		{"lo", arphrdLoopback, false, domain.InterfaceTypeLoopback},
		{"eth0", arphrdEther, false, domain.InterfaceTypeEthernet},
		{"wlan0", arphrdEther, true, domain.InterfaceTypeWiFi},
		{"mon0", arphrdIEEE80211Radiotap, false, domain.InterfaceTypeWiFi},
		{"ppp0", arphrdPPP, false, domain.InterfaceTypePPP},
		{"wg0", arphrdNone, false, domain.InterfaceTypeTunnel},
		{"sit0", arphrdSit, false, domain.InterfaceTypeTunnel},
		{"can0", 280, false, domain.InterfaceTypeOther},
	}
	for _, tt := range tests {
		if got := mapLinuxIfType(tt.arphrd, tt.name, tt.wireless); got != tt.want {
			t.Errorf("mapLinuxIfType(%d, %q, %v) = %d, want %d", tt.arphrd, tt.name, tt.wireless, got, tt.want)
		}
	}
}

func TestInterfacesOnThisHost(t *testing.T) {
	ifaces, err := NewLinuxSource().Interfaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, iface := range ifaces {
		// The settings view reads ipAddresses.length, so it must never be null.
		if iface.IPAddresses == nil {
			t.Errorf("%s: IPAddresses is nil", iface.Name)
		}
		if iface.Name == "lo" {
			if !iface.IsLoopback || iface.Type != domain.InterfaceTypeLoopback {
				t.Errorf("lo classified as %+v", iface)
			}
		}
	}
	data, err := json.Marshal(ifaces)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, m := range decoded {
		if m["ipAddresses"] == nil {
			t.Errorf("%v: ipAddresses serialised as null", m["name"])
		}
	}
}
