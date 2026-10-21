//go:build linux

package traffic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/netrasad/netrasad/internal/domain"
	"github.com/netrasad/netrasad/internal/platform/linux/netinfo"
)

const sysClassNet = "/sys/class/net"

// ARPHRD_* hardware types reported in /sys/class/net/<iface>/type.
const (
	arphrdEther             = 1
	arphrdPPP               = 512
	arphrdTunnel            = 768
	arphrdTunnel6           = 769
	arphrdLoopback          = 772
	arphrdSit               = 776
	arphrdIPGRE             = 778
	arphrdIEEE80211         = 801
	arphrdIEEE80211Prism    = 802
	arphrdIEEE80211Radiotap = 803
	arphrdIP6GRE            = 823
	arphrdNone              = 65534
)

const iffUp = 0x1

type LinuxSource struct {
	mu      sync.Mutex
	started bool
}

func NewLinuxSource() *LinuxSource {
	return &LinuxSource{}
}

func (l *LinuxSource) Start(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.started = true
	return nil
}

func (l *LinuxSource) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.started = false
	return nil
}

func (l *LinuxSource) Interfaces(ctx context.Context) ([]domain.NetworkInterface, error) {
	entries, err := os.ReadDir(sysClassNet)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", sysClassNet, err)
	}
	adapterInfo := netinfo.GetAdapterInfo()

	var ifaces []domain.NetworkInterface
	for _, entry := range entries {
		name := entry.Name()
		iface, err := readInterface(name)
		if err != nil {
			continue
		}
		if info, ok := adapterInfo[name]; ok {
			iface.IPAddresses = info.IPAddresses
			iface.GatewayIP = info.GatewayIP
			iface.SSID = info.SSID
		}
		ifaces = append(ifaces, iface)
	}
	return ifaces, nil
}

func (l *LinuxSource) ReadStats(ctx context.Context, interfaceID string) (domain.TrafficCounters, error) {
	rx, tx, err := readCounters(interfaceID)
	if err != nil {
		return domain.TrafficCounters{}, err
	}
	return domain.TrafficCounters{
		Timestamp:     time.Now(),
		InterfaceID:   interfaceID,
		DownloadBytes: rx,
		UploadBytes:   tx,
	}, nil
}

func (l *LinuxSource) ReadAllStats(ctx context.Context) ([]domain.TrafficCounters, error) {
	entries, err := os.ReadDir(sysClassNet)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var out []domain.TrafficCounters
	for _, entry := range entries {
		name := entry.Name()
		rx, tx, err := readCounters(name)
		if err != nil {
			continue
		}
		out = append(out, domain.TrafficCounters{
			Timestamp:     now,
			InterfaceID:   name,
			DownloadBytes: rx,
			UploadBytes:   tx,
		})
	}
	return out, nil
}

func readInterface(name string) (domain.NetworkInterface, error) {
	base := filepath.Join(sysClassNet, name)
	iface := domain.NetworkInterface{
		ID:          name,
		Name:        name,
		IPAddresses: []string{},
	}

	if desc, err := os.ReadFile(filepath.Join(base, "ifalias")); err == nil {
		iface.Description = strings.TrimSpace(string(desc))
	}
	if iface.Description == "" {
		if devPath, err := filepath.EvalSymlinks(filepath.Join(base, "device")); err == nil {
			iface.Description = filepath.Base(devPath)
		}
	}

	if idx, err := readUintFromFile(filepath.Join(base, "ifindex")); err == nil {
		iface.Index = uint32(idx)
	}

	if mac, err := os.ReadFile(filepath.Join(base, "address")); err == nil {
		iface.HardwareAddr = strings.TrimSpace(string(mac))
	}

	if tp, err := os.ReadFile(filepath.Join(base, "type")); err == nil {
		t, _ := strconv.Atoi(strings.TrimSpace(string(tp)))
		iface.Type = mapLinuxIfType(t, name, isWireless(base))
	}

	// carrier can only be read while the interface is administratively up.
	if carrier, err := readUintFromFile(filepath.Join(base, "carrier")); err == nil {
		iface.MediaConnected = carrier == 1
	}

	if operstate, err := os.ReadFile(filepath.Join(base, "operstate")); err == nil {
		switch strings.TrimSpace(string(operstate)) {
		case "up":
			iface.IsUp = true
		case "unknown":
			// Drivers that do not report link state (tun, ppp, some USB modems).
			iface.IsUp = hasFlagUp(name) && iface.MediaConnected
		}
	} else {
		iface.IsUp = hasFlagUp(name)
	}

	iface.IsLoopback = name == "lo" || iface.Type == domain.InterfaceTypeLoopback
	iface.IsVirtual = isLinuxVirtual(base, name)
	iface.IsVPN = isLinuxVPN(name)

	return iface, nil
}

func readCounters(ifaceID string) (rx, tx uint64, err error) {
	rx, err = readUintFromFile(filepath.Join(sysClassNet, ifaceID, "statistics", "rx_bytes"))
	if err != nil {
		return 0, 0, err
	}
	tx, err = readUintFromFile(filepath.Join(sysClassNet, ifaceID, "statistics", "tx_bytes"))
	if err != nil {
		return 0, 0, err
	}
	return rx, tx, nil
}

func readUintFromFile(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

func mapLinuxIfType(t int, name string, wireless bool) domain.InterfaceType {
	switch {
	case name == "lo" || t == arphrdLoopback:
		return domain.InterfaceTypeLoopback
	// Wi-Fi station interfaces report ARPHRD_ETHER, so check for wireless first.
	case wireless, t == arphrdIEEE80211, t == arphrdIEEE80211Prism, t == arphrdIEEE80211Radiotap:
		return domain.InterfaceTypeWiFi
	case t == arphrdEther:
		return domain.InterfaceTypeEthernet
	case t == arphrdPPP:
		return domain.InterfaceTypePPP
	case t == arphrdNone, t == arphrdTunnel, t == arphrdTunnel6, t == arphrdSit, t == arphrdIPGRE, t == arphrdIP6GRE:
		return domain.InterfaceTypeTunnel
	default:
		return domain.InterfaceTypeOther
	}
}

func isWireless(base string) bool {
	for _, p := range []string{"wireless", "phy80211"} {
		if _, err := os.Stat(filepath.Join(base, p)); err == nil {
			return true
		}
	}
	return false
}

func hasFlagUp(name string) bool {
	data, err := os.ReadFile(filepath.Join(sysClassNet, name, "flags"))
	if err != nil {
		return false
	}
	flags, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(string(data), "0x")), 16, 64)
	if err != nil {
		return false
	}
	return flags&iffUp != 0
}

func isLinuxVirtual(base, name string) bool {
	if _, err := os.Stat(filepath.Join(base, "device")); err != nil {
		return true
	}
	if strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "docker") ||
		strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "virbr") {
		return true
	}
	return false
}

func isLinuxVPN(name string) bool {
	lower := strings.ToLower(name)
	keywords := []string{"tun", "tap", "wg", "vpn", "ppp"}
	for _, kw := range keywords {
		if strings.HasPrefix(lower, kw) {
			return true
		}
	}
	return false
}
