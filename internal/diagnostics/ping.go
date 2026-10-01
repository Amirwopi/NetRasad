package diagnostics

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/netrasad/netrasad/internal/platform/executil"
)

func (s *Service) Ping(host string, count int) (*PingResult, error) {
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if count <= 0 {
		count = 4
	}

	var cmd *exec.Cmd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "ping", "-n", strconv.Itoa(count), host)
	case "linux":
		cmd = exec.CommandContext(ctx, "ping", "-c", strconv.Itoa(count), host)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	executil.HideWindow(cmd)

	out, err := cmd.CombinedOutput()
	output := string(out)
	result := &PingResult{
		Host: host,
		Sent: count,
		RTTs: []float64{},
	}

	if output != "" {
		result.Output = output
	}

	switch runtime.GOOS {
	case "windows":
		parseWindowsPing(result, output)
	case "linux":
		parseLinuxPing(result, output)
	}

	if result.Sent > 0 {
		result.PacketLoss = float64(result.Sent-result.Received) / float64(result.Sent) * 100
	}

	if len(result.RTTs) > 0 {
		var sum, mn, mx float64
		mn = result.RTTs[0]
		mx = result.RTTs[0]
		for _, r := range result.RTTs {
			sum += r
			if r < mn {
				mn = r
			}
			if r > mx {
				mx = r
			}
		}
		result.MinRTT = mn
		result.AvgRTT = sum / float64(len(result.RTTs))
		result.MaxRTT = mx
	}

	if err != nil && output == "" {
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	return result, nil
}

var windowsReplyRe = regexp.MustCompile(`time([=<])(\d+)\s*ms`)

var windowsStatsRe = regexp.MustCompile(`Minimum\s*=\s*(\d+)\s*ms.*Maximum\s*=\s*(\d+)\s*ms.*Average\s*=\s*(\d+)\s*ms`)

func parseWindowsPing(r *PingResult, output string) {
	matches := windowsReplyRe.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		if len(m) > 2 {
			op := m[1]
			if v, err := strconv.ParseFloat(m[2], 64); err == nil {
				if op == "<" {
					v = 0
				}
				r.RTTs = append(r.RTTs, v)
			}
		}
	}
	r.Received = len(r.RTTs)

	if sm := windowsStatsRe.FindStringSubmatch(output); len(sm) >= 4 {
		if v, err := strconv.ParseFloat(sm[1], 64); err == nil {
			r.MinRTT = v
		}
		if v, err := strconv.ParseFloat(sm[2], 64); err == nil {
			r.MaxRTT = v
		}
		if v, err := strconv.ParseFloat(sm[3], 64); err == nil {
			r.AvgRTT = v
		}
	}
}

var linuxReplyRe = regexp.MustCompile(`time=([\d.]+)\s*ms`)

func parseLinuxPing(r *PingResult, output string) {
	matches := linuxReplyRe.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		if len(m) > 1 {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil {
				r.RTTs = append(r.RTTs, v)
			}
		}
	}
	r.Received = len(r.RTTs)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "rtt ") || strings.HasPrefix(line, "round-trip ") {
			if idx := strings.Index(line, "="); idx >= 0 {
				statsPart := strings.TrimSpace(line[idx+1:])
				statsPart = strings.TrimSuffix(statsPart, " ms")
				parts := strings.Split(statsPart, "/")
				if len(parts) >= 3 {
					if v, err := strconv.ParseFloat(parts[0], 64); err == nil {
						r.MinRTT = v
					}
					if v, err := strconv.ParseFloat(parts[1], 64); err == nil {
						r.AvgRTT = v
					}
					if v, err := strconv.ParseFloat(parts[2], 64); err == nil {
						r.MaxRTT = v
					}
				}
			}
		}
	}
}
