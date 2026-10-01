package traffic

import (
	"testing"

	"github.com/netrasad/netrasad/internal/domain"
)

func TestClassifier_ShouldCountInterface(t *testing.T) {
	c := NewClassifier(ClassificationRules{
		IgnoreLoopback:   true,
		IgnoreInterfaces: []string{"luid-bad"},
		OnlyInterfaces:   []string{"luid-good1", "luid-good2"},
	})

	tests := []struct {
		id       string
		loopback bool
		want     bool
	}{
		{"luid-good1", false, true},
		{"luid-good2", false, true},
		{"luid-good1", true, false},  // loopback ignored
		{"luid-bad", false, false},   // in ignore list
		{"luid-other", false, false}, // not in only list
	}
	for _, tt := range tests {
		got := c.ShouldCountInterface(tt.id, tt.loopback)
		if got != tt.want {
			t.Errorf("ShouldCountInterface(%q, loopback=%v) = %v, want %v", tt.id, tt.loopback, got, tt.want)
		}
	}
}

func TestClassifier_ClassifyAddress(t *testing.T) {
	c := NewClassifier(ClassificationRules{
		IgnoreLoopback: true,
		IgnoreLAN:      false,
	})

	tests := []struct {
		ip   string
		want domain.TrafficCategory
	}{
		{"8.8.8.8", domain.CategoryInternet},
		{"1.1.1.1", domain.CategoryInternet},
		{"192.168.1.1", domain.CategoryLAN},
		{"10.0.0.1", domain.CategoryLAN},
		{"172.16.5.5", domain.CategoryLAN},
		{"127.0.0.1", domain.CategoryIgnored}, // loopback ignored
		{"169.254.1.1", domain.CategoryLAN},
		{"::1", domain.CategoryIgnored}, // loopback ignored
		{"fe80::1", domain.CategoryLAN},
		{"fd00::1", domain.CategoryLAN},
	}
	for _, tt := range tests {
		got := c.ClassifyAddress(tt.ip)
		if got != tt.want {
			t.Errorf("ClassifyAddress(%q) = %d, want %d", tt.ip, got, tt.want)
		}
	}
}

func TestClassifier_ClassifyAddress_IgnoreLAN(t *testing.T) {
	c := NewClassifier(ClassificationRules{
		IgnoreLAN: true,
	})

	if got := c.ClassifyAddress("192.168.1.1"); got != domain.CategoryIgnored {
		t.Errorf("ClassifyAddress(192.168.1.1) with IgnoreLAN = %d, want %d (ignored)", got, domain.CategoryIgnored)
	}
	if got := c.ClassifyAddress("8.8.8.8"); got != domain.CategoryInternet {
		t.Errorf("ClassifyAddress(8.8.8.8) with IgnoreLAN = %d, want %d (internet)", got, domain.CategoryInternet)
	}
}

func TestClassifier_CustomCIDRs(t *testing.T) {
	c := NewClassifier(ClassificationRules{
		CustomLANCIDRs: []string{"100.64.0.0/10"}, // CGNAT range
	})
	if got := c.ClassifyAddress("100.64.0.1"); got != domain.CategoryLAN {
		t.Errorf("ClassifyAddress(100.64.0.1) with custom CIDR = %d, want %d (LAN)", got, domain.CategoryLAN)
	}
}

func TestClassifier_IgnoreCIDRs(t *testing.T) {
	c := NewClassifier(ClassificationRules{
		IgnoreCIDRs: []string{"8.8.8.0/24"},
	})
	if got := c.ClassifyAddress("8.8.8.8"); got != domain.CategoryIgnored {
		t.Errorf("ClassifyAddress(8.8.8.8) with ignore CIDR = %d, want %d (ignored)", got, domain.CategoryIgnored)
	}
}

func TestClassifier_InvalidCIDRsSkipped(t *testing.T) {
	c := NewClassifier(ClassificationRules{
		CustomLANCIDRs: []string{"not-a-cidr", "10.0.0.0/8"},
	})
	// Valid one should still work.
	if got := c.ClassifyAddress("10.1.2.3"); got != domain.CategoryLAN {
		t.Errorf("valid CIDR should still classify, got %d", got)
	}
}

func TestClassifier_ClassifyInterface(t *testing.T) {
	c := NewClassifier(ClassificationRules{IgnoreLoopback: true})

	loopback := domain.NetworkInterface{IsLoopback: true}
	if got := c.ClassifyInterface(loopback); got != domain.CategoryIgnored {
		t.Errorf("loopback interface with IgnoreLoopback = %d, want %d", got, domain.CategoryIgnored)
	}

	physical := domain.NetworkInterface{IsLoopback: false}
	if got := c.ClassifyInterface(physical); got != domain.CategoryInternet {
		t.Errorf("physical interface = %d, want %d", got, domain.CategoryInternet)
	}
}
