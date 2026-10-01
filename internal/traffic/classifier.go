package traffic

import (
	"net"
	"strings"
	"sync"

	"github.com/netrasad/netrasad/internal/domain"
)

type ClassificationRules struct {
	IgnoreLAN bool
	IgnoreLoopback bool
	IgnoreCIDRs []string
	CustomLANCIDRs []string
	OnlyInterfaces []string
	IgnoreInterfaces []string
}

var DefaultPrivateCIDRs = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"fc00::/7",
	"fe80::/10",
}

type Classifier struct {
	rules        ClassificationRules
	privateNets  []*net.IPNet
	ignoreNets   []*net.IPNet
	ignoreIfaces map[string]bool
	onlyIfaces   map[string]bool
	mu           sync.RWMutex
}

func NewClassifier(rules ClassificationRules) *Classifier {
	c := &Classifier{
		rules:        rules,
		ignoreIfaces: toSet(rules.IgnoreInterfaces),
		onlyIfaces:   toSet(rules.OnlyInterfaces),
	}
	allPrivate := append(append([]string{}, DefaultPrivateCIDRs...), rules.CustomLANCIDRs...)
	c.privateNets = parseCIDRs(allPrivate)
	c.ignoreNets = parseCIDRs(rules.IgnoreCIDRs)
	return c
}

func (c *Classifier) ShouldCountInterface(ifaceID string, isLoopback bool) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.ignoreIfaces[ifaceID] {
		return false
	}
	if isLoopback && c.rules.IgnoreLoopback {
		return false
	}
	if len(c.onlyIfaces) > 0 && !c.onlyIfaces[ifaceID] {
		return false
	}
	return true
}

func (c *Classifier) ClassifyAddress(ipStr string) domain.TrafficCategory {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return domain.CategoryInternet
	}
	if ip.IsLoopback() {
		if c.rules.IgnoreLoopback {
			return domain.CategoryIgnored
		}
		return domain.CategoryLoopback
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		if c.rules.IgnoreLAN {
			return domain.CategoryIgnored
		}
		return domain.CategoryLAN
	}
	if c.inAnyNet(ip, c.ignoreNets) {
		return domain.CategoryIgnored
	}
	if c.inAnyNet(ip, c.privateNets) {
		if c.rules.IgnoreLAN {
			return domain.CategoryIgnored
		}
		return domain.CategoryLAN
	}
	return domain.CategoryInternet
}

func (c *Classifier) ClassifyInterface(iface domain.NetworkInterface) domain.TrafficCategory {
	if iface.IsLoopback {
		if c.rules.IgnoreLoopback {
			return domain.CategoryIgnored
		}
		return domain.CategoryLoopback
	}
	return domain.CategoryInternet
}

func (c *Classifier) inAnyNet(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func parseCIDRs(cidrs []string) []*net.IPNet {
	var nets []*net.IPNet
	for _, s := range cidrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			continue
		}
		nets = append(nets, n)
	}
	return nets
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, s := range items {
		m[s] = true
	}
	return m
}

func (c *Classifier) SetOnlyInterfaces(ids []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rules.OnlyInterfaces = ids
	c.onlyIfaces = toSet(ids)
}

func (c *Classifier) GetOnlyInterfaces() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, len(c.rules.OnlyInterfaces))
	copy(out, c.rules.OnlyInterfaces)
	return out
}
