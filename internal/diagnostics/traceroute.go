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

func (s *Service) Traceroute(host string, maxHops int) (*TracerouteResult, error) {
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if maxHops <= 0 {
		maxHops = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "tracert", "-d", "-h", strconv.Itoa(maxHops), host)
	case "linux":
		cmd = exec.CommandContext(ctx, "traceroute", "-n", "-m", strconv.Itoa(maxHops), host)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	executil.HideWindow(cmd)

	out, err := cmd.CombinedOutput()
	output := string(out)
	result := &TracerouteResult{
		Hops:   []TracerouteHop{},
		Output: output,
	}

	switch runtime.GOOS {
	case "windows":
		parseWindowsTracert(result, output)
	case "linux":
		parseLinuxTraceroute(result, output)
	}

	if err != nil && output == "" {
		return nil, fmt.Errorf("traceroute failed: %w", err)
	}

	return result, nil
}

var windowsTracertRe = regexp.MustCompile(`^\s*(\d+)\s+(.+?)\s+(\S+)\s*$`)

func parseWindowsTracert(r *TracerouteResult, output string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !isDigit(rune(line[0])) {
			continue
		}

		hop := parseWindowsTracertLine(line)
		if hop != nil {
			r.Hops = append(r.Hops, *hop)
		}
	}
}

func parseWindowsTracertLine(line string) *TracerouteHop {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return nil
	}
	hopNum, err := strconv.Atoi(fields[0])
	if err != nil {
		return nil
	}

	hop := &TracerouteHop{
		Hop:  hopNum,
		RTT1: -1,
		RTT2: -1,
		RTT3: -1,
	}

	ipField := fields[len(fields)-1]
	hop.IP = ipField

	rttFields := fields[1 : len(fields)-1]
	if len(rttFields) >= 3 {
		hop.RTT1 = parseWindowsRTT(rttFields[0])
		hop.RTT2 = parseWindowsRTT(rttFields[1])
		hop.RTT3 = parseWindowsRTT(rttFields[2])
	}

	return hop
}

func parseWindowsRTT(s string) float64 {
	if s == "*" {
		return -1
	}
	if strings.HasPrefix(s, "<") {
		return 0
	}
	if s == "ms" {
		return -1
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1
	}
	return v
}

var linuxTracerouteRe = regexp.MustCompile(`^\s*(\d+)\s+(.*)$`)

func parseLinuxTraceroute(r *TracerouteResult, output string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !isDigit(rune(line[0])) {
			continue
		}

		hop := parseLinuxTracerouteLine(line)
		if hop != nil {
			r.Hops = append(r.Hops, *hop)
		}
	}
}

func parseLinuxTracerouteLine(line string) *TracerouteHop {
	m := linuxTracerouteRe.FindStringSubmatch(line)
	if len(m) < 3 {
		return nil
	}
	hopNum, err := strconv.Atoi(m[1])
	if err != nil {
		return nil
	}

	rest := strings.TrimSpace(m[2])
	hop := &TracerouteHop{
		Hop:  hopNum,
		RTT1: -1,
		RTT2: -1,
		RTT3: -1,
	}

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return hop
	}

	if fields[0] == "*" {
		return hop
	}

	hop.IP = fields[0]
	hop.Host = fields[0]

	rtts := []float64{-1, -1, -1}
	rttIdx := 0
	for i := 1; i < len(fields) && rttIdx < 3; i++ {
		f := fields[i]
		if f == "ms" {
			continue
		}
		if f == "*" {
			rttIdx++
			continue
		}
		v, err := strconv.ParseFloat(f, 64)
		if err == nil {
			if rttIdx < 3 {
				rtts[rttIdx] = v
			}
		}
		rttIdx++
	}
	hop.RTT1 = rtts[0]
	hop.RTT2 = rtts[1]
	hop.RTT3 = rtts[2]

	return hop
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
