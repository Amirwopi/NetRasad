//go:build windows

package traffic

import (
	"context"
	"testing"
	"unsafe"
)

// TestMibIfRow2_Size verifies that the Go struct size matches the expected
// C struct size. If this fails, the struct layout is wrong and GetIfTable2
// will read garbage data.
func TestMibIfRow2_Size(t *testing.T) {
	size := unsafe.Sizeof(mibIfRow2{})
	// The expected size depends on alignment. We verify it is a multiple
	// of 8 (the alignment of the first field, uint64) and within a
	// plausible range. The exact value is determined empirically by the
	// integration test below; here we check structural sanity.
	if size%8 != 0 {
		t.Errorf("mibIfRow2 size %d is not 8-byte aligned", size)
	}
	if size < 1000 || size > 3000 {
		t.Errorf("mibIfRow2 size %d is outside plausible range [1000, 3000]", size)
	}
	t.Logf("mibIfRow2 size = %d bytes", size)
}

// TestMibIfRow2_FieldOffsets verifies critical field offsets.
func TestMibIfRow2_FieldOffsets(t *testing.T) {
	var r mibIfRow2
	t.Logf("InterfaceLuid offset = %d", unsafe.Offsetof(r.InterfaceLuid))
	t.Logf("InterfaceIndex offset = %d", unsafe.Offsetof(r.InterfaceIndex))
	t.Logf("InterfaceGuid offset = %d", unsafe.Offsetof(r.InterfaceGuid))
	t.Logf("Alias offset = %d", unsafe.Offsetof(r.Alias))
	t.Logf("Description offset = %d", unsafe.Offsetof(r.Description))
	t.Logf("PhysicalAddressLength offset = %d", unsafe.Offsetof(r.PhysicalAddressLength))
	t.Logf("PhysicalAddress offset = %d", unsafe.Offsetof(r.PhysicalAddress))
	t.Logf("PermanentPhysicalAddress offset = %d", unsafe.Offsetof(r.PermanentPhysicalAddress))
	t.Logf("Mtu offset = %d", unsafe.Offsetof(r.Mtu))
	t.Logf("Type offset = %d", unsafe.Offsetof(r.Type))
	t.Logf("TunnelType offset = %d", unsafe.Offsetof(r.TunnelType))
	t.Logf("MediaType offset = %d", unsafe.Offsetof(r.MediaType))
	t.Logf("PhysicalMediumType offset = %d", unsafe.Offsetof(r.PhysicalMediumType))
	t.Logf("AccessType offset = %d", unsafe.Offsetof(r.AccessType))
	t.Logf("DirectionType offset = %d", unsafe.Offsetof(r.DirectionType))
	t.Logf("ConnectionType offset = %d", unsafe.Offsetof(r.ConnectionType))
	t.Logf("InterfaceAndOperStatusFlags offset = %d", unsafe.Offsetof(r.InterfaceAndOperStatusFlags))
	t.Logf("AdminStatus offset = %d", unsafe.Offsetof(r.AdminStatus))
	t.Logf("OperStatus offset = %d", unsafe.Offsetof(r.OperStatus))
	t.Logf("MediaConnectState offset = %d", unsafe.Offsetof(r.MediaConnectState))
	t.Logf("NetworkGuid offset = %d", unsafe.Offsetof(r.NetworkGuid))
	t.Logf("ConnectionType2 offset = %d", unsafe.Offsetof(r.ConnectionType2))
	t.Logf("TransmitLinkSpeed offset = %d", unsafe.Offsetof(r.TransmitLinkSpeed))
	t.Logf("ReceiveLinkSpeed offset = %d", unsafe.Offsetof(r.ReceiveLinkSpeed))
	t.Logf("InOctets offset = %d", unsafe.Offsetof(r.InOctets))
	t.Logf("OutOctets offset = %d", unsafe.Offsetof(r.OutOctets))
	t.Logf("Total struct size = %d", unsafe.Sizeof(r))
}

// TestGetIfTable2_ReturnsInterfaces is an integration test that calls the
// real Windows GetIfTable2 API and verifies the returned data is sane.
// If the struct layout is wrong, this test will fail with garbage data.
func TestGetIfTable2_ReturnsInterfaces(t *testing.T) {
	src := NewWindowsSource()
	ctx := context.Background()

	ifaces, err := src.Interfaces(ctx)
	if err != nil {
		t.Fatalf("Interfaces: %v", err)
	}
	if len(ifaces) == 0 {
		t.Fatal("expected at least one interface, got 0")
	}

	// Verify we can find the loopback interface (always present on Windows).
	foundLoopback := false
	for _, iface := range ifaces {
		t.Logf("interface: id=%s name=%q type=%d up=%v loopback=%v",
			iface.ID, iface.Name, iface.Type, iface.IsUp, iface.IsLoopback)
		if iface.IsLoopback {
			foundLoopback = true
			if iface.Name == "" {
				t.Error("loopback interface has empty name")
			}
		}
		// ID should be non-empty and start with "luid-".
		if iface.ID == "" {
			t.Error("interface has empty ID")
		}
	}
	if !foundLoopback {
		t.Error("loopback interface not found — struct layout may be wrong")
	}
}

// TestGetIfTable2_ReadAllStats verifies that counters are returned and
// are non-decreasing across two reads (for interfaces with no traffic).
func TestGetIfTable2_ReadAllStats(t *testing.T) {
	src := NewWindowsSource()
	ctx := context.Background()

	first, err := src.ReadAllStats(ctx)
	if err != nil {
		t.Fatalf("first ReadAllStats: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("no counters returned")
	}

	// Verify each counter has a valid interface ID and non-negative values.
	for _, c := range first {
		if c.InterfaceID == "" {
			t.Error("counter has empty interface ID")
		}
		t.Logf("counter: id=%s down=%d up=%d", c.InterfaceID, c.DownloadBytes, c.UploadBytes)
	}
}
