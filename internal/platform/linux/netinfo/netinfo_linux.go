//go:build linux

// Package netinfo collects per-interface addressing details on Linux: unicast
// addresses, the IPv4 default gateway and the SSID of connected Wi-Fi links.
// It is the Linux counterpart of platform/windows/netinfo.
package netinfo

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

type AdapterInfo struct {
	IPAddresses []string
	GatewayIP   string
	SSID        string
}

type Route struct {
	Interface string
	Gateway   string
	Metric    int
}

const (
	rtfUp      = 0x1
	rtfGateway = 0x2
)

// GetAdapterInfo returns addressing details keyed by interface name.
func GetAdapterInfo() map[string]AdapterInfo {
	out := map[string]AdapterInfo{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	addrs, _ := addressesByIndex()

	gateways := map[string]string{}
	if routes, err := DefaultRoutes(); err == nil {
		for _, r := range routes {
			if _, ok := gateways[r.Interface]; !ok {
				gateways[r.Interface] = r.Gateway
			}
		}
	}

	ssids := wifiSSIDs()
	for _, ifi := range ifaces {
		out[ifi.Name] = AdapterInfo{
			IPAddresses: formatIPs(addrs[ifi.Index]),
			GatewayIP:   gateways[ifi.Name],
			SSID:        ssids[ifi.Name],
		}
	}
	return out
}

// DefaultRoutes returns the IPv4 default routes of the main routing table that
// go through a gateway, lowest metric first.
func DefaultRoutes() ([]Route, error) {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseDefaultRoutes(f)
}

func parseDefaultRoutes(r io.Reader) ([]Route, error) {
	var routes []Route
	sc := bufio.NewScanner(r)
	header := true
	for sc.Scan() {
		if header {
			header = false
			continue
		}
		// Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT
		f := strings.Fields(sc.Text())
		if len(f) < 8 || f[1] != "00000000" || f[7] != "00000000" {
			continue
		}
		flags, err := strconv.ParseUint(f[3], 16, 32)
		if err != nil || flags&rtfUp == 0 || flags&rtfGateway == 0 {
			continue
		}
		gw, err := parseHexIPv4(f[2])
		if err != nil {
			continue
		}
		metric, _ := strconv.Atoi(f[6])
		routes = append(routes, Route{Interface: f[0], Gateway: gw, Metric: metric})
	}
	sort.SliceStable(routes, func(i, j int) bool { return routes[i].Metric < routes[j].Metric })
	return routes, sc.Err()
}

// parseHexIPv4 decodes an address from /proc/net/route, which prints the raw
// network-order bytes as a host-endian hex integer.
func parseHexIPv4(s string) (string, error) {
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return "", err
	}
	var b [4]byte
	binary.NativeEndian.PutUint32(b[:], uint32(v))
	return net.IP(b[:]).String(), nil
}

// addressesByIndex lists every interface address with a single rtnetlink dump
// instead of one dump per interface as net.Interface.Addrs does.
func addressesByIndex() (map[int][]net.IP, error) {
	rib, err := syscall.NetlinkRIB(syscall.RTM_GETADDR, syscall.AF_UNSPEC)
	if err != nil {
		return nil, err
	}
	msgs, err := syscall.ParseNetlinkMessage(rib)
	if err != nil {
		return nil, err
	}
	out := make(map[int][]net.IP)
	for i := range msgs {
		m := &msgs[i]
		if m.Header.Type == syscall.NLMSG_DONE {
			break
		}
		if m.Header.Type != syscall.RTM_NEWADDR || len(m.Data) < syscall.SizeofIfAddrmsg {
			continue
		}
		attrs, err := syscall.ParseNetlinkRouteAttr(m)
		if err != nil {
			continue
		}
		var addr, local net.IP
		for _, a := range attrs {
			switch a.Attr.Type {
			case syscall.IFA_ADDRESS:
				addr = append(net.IP(nil), a.Value...)
			case syscall.IFA_LOCAL:
				local = append(net.IP(nil), a.Value...)
			}
		}
		// On IPv4 point-to-point links IFA_ADDRESS is the peer and IFA_LOCAL ours.
		family := m.Data[0]
		if family == syscall.AF_INET && local != nil {
			addr = local
		}
		if addr == nil {
			continue
		}
		index := int(binary.NativeEndian.Uint32(m.Data[4:8]))
		out[index] = append(out[index], addr)
	}
	return out, nil
}

// formatIPs renders addresses with IPv4 first so the most recognisable one
// leads the list.
func formatIPs(ips []net.IP) []string {
	out := make([]string, 0, len(ips))
	for _, wantV4 := range []bool{true, false} {
		for _, ip := range ips {
			if (ip.To4() != nil) == wantV4 {
				out = append(out, ip.String())
			}
		}
	}
	return out
}
