//go:build windows

package traffic

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/netrasad/netrasad/internal/domain"
	"github.com/netrasad/netrasad/internal/platform/windows/netinfo"
	"golang.org/x/sys/windows"
)

var (
	iphlpapi         = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetIfTable2  = iphlpapi.NewProc("GetIfTable2")
	procFreeMibTable = iphlpapi.NewProc("FreeMibTable")
)

const (
	ifMaxStringSize        = 256
	ifMaxPhysAddrLength    = 32
	ifOperStatusUp         = 1
	ifTypeSoftwareLoopback = 24
)

type ifOperStatusFlags struct {
	FilterPassThru uint8
	FilterHardware uint8
	FilterEndPoint uint8
	FilterHyperV   uint8
}

type mibIfRow2 struct {
	InterfaceLuid               uint64
	InterfaceIndex              uint32
	InterfaceGuid               windows.GUID
	Alias                       [ifMaxStringSize + 1]uint16
	Description                 [ifMaxStringSize + 1]uint16
	PhysicalAddressLength       uint32
	PhysicalAddress             [ifMaxPhysAddrLength]byte
	PermanentPhysicalAddress    [ifMaxPhysAddrLength]byte
	Mtu                         uint32
	Type                        uint32
	TunnelType                  uint32
	MediaType                   uint32
	PhysicalMediumType          uint32
	AccessType                  uint32
	DirectionType               uint32
	ConnectionType              uint32
	InterfaceAndOperStatusFlags ifOperStatusFlags
	AdminStatus                 uint32
	OperStatus                  uint32
	MediaConnectState           uint32
	NetworkGuid                 windows.GUID
	AliasIfKey                  windows.GUID
	ConnectionType2             uint32
	pad0                        uint32
	TransmitLinkSpeed           uint64
	ReceiveLinkSpeed            uint64
	InUcastOctets               uint64
	InMulticastOctets           uint64
	InBroadcastOctets           uint64
	InDiscards                  uint64
	InErrors                    uint64
	InUnknownProtos             uint64
	InOctets                    uint64
	InUcastPkts                 uint64
	InNUcastPkts                uint64
	OutUcastOctets              uint64
	OutMulticastOctets          uint64
	OutOctets                   uint64
	OutUcastPkts                uint64
	OutNUcastPkts               uint64
	OutDiscards                 uint64
}

type mibIfTable2 struct {
	NumEntries uint32
}

type WindowsSource struct {
	mu      sync.Mutex
	started bool
}

func NewWindowsSource() *WindowsSource {
	return &WindowsSource{}
}

func (w *WindowsSource) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.started = true
	return nil
}

func (w *WindowsSource) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.started = false
	return nil
}

func (w *WindowsSource) getIfTable2() ([]mibIfRow2, error) {
	var tablePtr unsafe.Pointer
	r1, _, err := procGetIfTable2.Call(uintptr(unsafe.Pointer(&tablePtr)))
	if r1 != 0 {
		return nil, fmt.Errorf("GetIfTable2 failed: %w (code %d)", err, r1)
	}
	if tablePtr == nil {
		return nil, fmt.Errorf("GetIfTable2 returned null pointer")
	}
	defer procFreeMibTable.Call(uintptr(tablePtr))

	numEntries := *(*uint32)(tablePtr)
	if numEntries == 0 {
		return []mibIfRow2{}, nil
	}

	const arrayOffset = 8

	rows := unsafe.Slice((*mibIfRow2)(unsafe.Add(tablePtr, arrayOffset)), numEntries)

	out := make([]mibIfRow2, numEntries)
	copy(out, rows)
	return out, nil
}

func (w *WindowsSource) Interfaces(ctx context.Context) ([]domain.NetworkInterface, error) {
	rows, err := w.getIfTable2()
	if err != nil {
		return nil, err
	}

	adapterInfo := netinfo.GetAdapterInfo()

	ifaces := make([]domain.NetworkInterface, 0, len(rows))
	for _, row := range rows {
		iface := rowToInterface(row)
		if info, ok := adapterInfo[row.InterfaceLuid]; ok {
			if len(info.IPAddresses) > 0 {
				iface.IPAddresses = info.IPAddresses
			}
			iface.GatewayIP = info.GatewayIP
			iface.SSID = info.SSID
		}
		ifaces = append(ifaces, iface)
	}
	return ifaces, nil
}

func (w *WindowsSource) ReadStats(ctx context.Context, interfaceID string) (domain.TrafficCounters, error) {
	rows, err := w.getIfTable2()
	if err != nil {
		return domain.TrafficCounters{}, err
	}
	for _, row := range rows {
		id := interfaceIDFromRow(row)
		if id == interfaceID {
			return rowToCounters(row), nil
		}
	}
	return domain.TrafficCounters{}, fmt.Errorf("interface %q not found", interfaceID)
}

func (w *WindowsSource) ReadAllStats(ctx context.Context) ([]domain.TrafficCounters, error) {
	rows, err := w.getIfTable2()
	if err != nil {
		return nil, err
	}
	out := make([]domain.TrafficCounters, 0, len(rows))
	now := time.Now()
	for _, row := range rows {
		c := rowToCounters(row)
		c.Timestamp = now
		out = append(out, c)
	}
	return out, nil
}

func rowToInterface(row mibIfRow2) domain.NetworkInterface {
	iface := domain.NetworkInterface{
		ID:             interfaceIDFromRow(row),
		Name:           utf16ToString(row.Alias[:]),
		Description:    utf16ToString(row.Description[:]),
		Index:          row.InterfaceIndex,
		Type:           domain.InterfaceType(row.Type),
		IsUp:           row.OperStatus == ifOperStatusUp || row.MediaConnectState == 1,
		IsLoopback:     row.Type == ifTypeSoftwareLoopback,
		MediaConnected: row.MediaConnectState != 2,
		IPAddresses:    []string{},
	}
	iface.HardwareAddr = formatMAC(row.PhysicalAddress[:row.PhysicalAddressLength])
	iface.IsVirtual = isVirtualInterface(row)
	iface.IsVPN = isVPNInterface(row, iface.Name, iface.Description)
	return iface
}

func rowToCounters(row mibIfRow2) domain.TrafficCounters {
	return domain.TrafficCounters{
		InterfaceID:   interfaceIDFromRow(row),
		DownloadBytes: row.InOctets,
		UploadBytes:   row.OutOctets,
		Timestamp:     time.Now(),
	}
}

func interfaceIDFromRow(row mibIfRow2) string {
	return fmt.Sprintf("luid-%016x", row.InterfaceLuid)
}

func utf16ToString(buf []uint16) string {
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return windows.UTF16ToString(buf[:n])
}

func formatMAC(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	parts := make([]string, len(b))
	for i, v := range b {
		parts[i] = fmt.Sprintf("%02x", v)
	}
	return strings.Join(parts, ":")
}

func isVirtualInterface(row mibIfRow2) bool {
	switch row.Type {
	case ifTypeSoftwareLoopback:
		return true
	case 131:
		return true
	case 53:
		return true
	}
	name := strings.ToLower(utf16ToString(row.Alias[:]) + " " + utf16ToString(row.Description[:]))
	filterKeywords := []string{"wfp", "qos", "lightweight filter", "winpk", "packet scheduler", "filter driver"}
	for _, kw := range filterKeywords {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}

func isVPNInterface(row mibIfRow2, name, desc string) bool {
	if row.MediaType == 0x00000006 {
		return true
	}
	lower := strings.ToLower(name + " " + desc)
	vpnKeywords := []string{"vpn", "wireguard", "openvpn", "tunnel", "tap-", "tun"}
	for _, kw := range vpnKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
