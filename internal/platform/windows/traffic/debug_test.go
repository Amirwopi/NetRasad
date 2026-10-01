//go:build windows

package traffic

import (
	"context"
	"testing"
)

func TestDebugAllInterfaces(t *testing.T) {
	src := NewWindowsSource()
	if err := src.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer src.Stop()

	ifaces, err := src.Interfaces(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("=== Total interfaces: %d ===", len(ifaces))
	for i, iface := range ifaces {
		active := iface.IsUp && !iface.IsLoopback && !iface.IsVirtual && iface.MediaConnected
		t.Logf("[%d] id=%s name=%q type=%d isUp=%v isLoopback=%v isVirtual=%v isVPN=%v mediaConnected=%v ACTIVE=%v gateway=%s ssid=%q ips=%v",
			i, iface.ID, iface.Name, iface.Type, iface.IsUp, iface.IsLoopback, iface.IsVirtual, iface.IsVPN, iface.MediaConnected, active, iface.GatewayIP, iface.SSID, iface.IPAddresses)
	}

	// Count active
	activeCount := 0
	for _, iface := range ifaces {
		if iface.IsUp && !iface.IsLoopback && !iface.IsVirtual && iface.MediaConnected {
			activeCount++
		}
	}
	t.Logf("=== Active interfaces: %d ===", activeCount)

	if activeCount == 0 {
		t.Log("No active connected physical interfaces detected at test time (possibly offline/disconnected).")
	}
}
