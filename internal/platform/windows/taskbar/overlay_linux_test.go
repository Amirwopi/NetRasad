//go:build linux

package taskbar

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestStatusFollowsEnabledFlag(t *testing.T) {
	o := &TaskbarOverlay{downRate: 2 * 1024 * 1024, upRate: 512}
	o.UpdateTopProcesses([]TopProcessEntry{
		{Name: "firefox", DownloadBps: 1536, UploadBps: 10},
		{Name: "a", DownloadBps: 1}, {Name: "b"}, {Name: "c"},
	})

	label, guide, tip := o.status()
	if label != "" || guide != "" {
		t.Errorf("disabled overlay label = %q / %q, want empty", label, guide)
	}
	if !strings.HasPrefix(tip, "▼ 16.0 Mbit   ▲ 512 B/s") || !strings.Contains(tip, "firefox   ▼ 1.5 KB/s ▲ 10 B/s") {
		t.Errorf("tooltip = %q", tip)
	}
	if strings.Contains(tip, "\nc ") {
		t.Error("tooltip must list at most three applications")
	}

	o.SetEnabled(true)
	if label, guide, _ = o.status(); label != "▼ 16.0 Mbit ▲ 512 B/s" || guide != labelGuide {
		t.Errorf("enabled overlay label = %q / %q", label, guide)
	}
	if !o.IsEnabled() {
		t.Error("IsEnabled() = false after SetEnabled(true)")
	}
}

func TestLabelGuideIsWidestLabel(t *testing.T) {
	widest := "▼ " + formatRateMbit(1024*1024-1) + " ▲ " + formatRateMbit(1024*1024-1)
	if utf8.RuneCountInString(widest) > utf8.RuneCountInString(labelGuide) {
		t.Errorf("label %q is wider than the guide %q", widest, labelGuide)
	}
}

func TestTruncateStrKeepsRunesIntact(t *testing.T) {
	got := truncateStr("برنامه‌ی-بسیار-طولانی-برای-آزمایش-برش", 10)
	if !utf8.ValidString(got) || utf8.RuneCountInString(got) != 10 || !strings.HasSuffix(got, "…") {
		t.Errorf("truncateStr = %q", got)
	}
	if got := truncateStr("short", 25); got != "short" {
		t.Errorf("truncateStr(short) = %q", got)
	}
}
