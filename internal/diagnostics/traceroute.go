package diagnostics

import (
	"context"
	"fmt"
	"net"
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
	var parse func(*TracerouteResult, string)
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "tracert", "-d", "-h", strconv.Itoa(maxHops), host)
		parse = parseWindowsTracert
	case "linux":
		// traceroute is not installed by default on several distributions,
		// while tracepath ships with iputils next to ping.
		if _, err := exec.LookPath("traceroute"); err == nil {
			cmd = exec.CommandContext(ctx, "traceroute", "-n", "-m", strconv.Itoa(maxHops), host)
			parse = parseLinuxTraceroute
		} else if _, err := exec.LookPath("tracepath"); err == nil {
			cmd = exec.CommandContext(ctx, "tracepath", "-n", "-m", strconv.Itoa(maxHops), host)
			parse = parseTracepath
		} else {
			return nil, fmt.Errorf("traceroute failed: install the traceroute or iputils (tracepath) package")
		}
		cmd.Env = untranslatedEnv()
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

	parse(result, output)

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

	// Each probe prints either "*" or "<rtt> ms", optionally preceded by the
	// replying address (repeated when it changes) and followed by annotations
	// such as "!H".
	rtts := []float64{-1, -1, -1}
	probe := 0
	fields := strings.Fields(rest)
	for i := 0; i < len(fields) && probe < 3; i++ {
		f := fields[i]
		switch {
		case f == "*":
			probe++
		case i+1 < len(fields) && fields[i+1] == "ms":
			if v, err := strconv.ParseFloat(f, 64); err == nil {
				rtts[probe] = v
			}
			probe++
			i++
		case strings.HasSuffix(f, "ms"):
			if v, err := strconv.ParseFloat(strings.TrimSuffix(f, "ms"), 64); err == nil {
				rtts[probe] = v
				probe++
			}
		case hop.IP == "" && net.ParseIP(strings.Trim(f, "()")) != nil:
			hop.IP = strings.Trim(f, "()")
			hop.Host = hop.IP
		}
	}
	hop.RTT1 = rtts[0]
	hop.RTT2 = rtts[1]
	hop.RTT3 = rtts[2]

	return hop
}

var tracepathLineRe = regexp.MustCompile(`^\s*(\d+)\??:\s+(.*)$`)

// parseTracepath handles iputils tracepath, which prints one line per probe:
//
//	1?: [LOCALHOST]                      pmtu 1500
//	1:  192.168.1.1                                           0.523ms
//	2:  no reply
//	3:  8.8.8.8                                              10.904ms reached
func parseTracepath(r *TracerouteResult, output string) {
	hops := map[int]*TracerouteHop{}
	var order []int
	for _, line := range strings.Split(output, "\n") {
		m := tracepathLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		fields := strings.Fields(m[2])
		if len(fields) == 0 || fields[0] == "[LOCALHOST]" {
			continue
		}
		num, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		hop, ok := hops[num]
		if !ok {
			hop = &TracerouteHop{Hop: num, RTT1: -1, RTT2: -1, RTT3: -1}
			hops[num] = hop
			order = append(order, num)
		}
		if net.ParseIP(fields[0]) == nil {
			continue
		}
		if hop.IP == "" {
			hop.IP = fields[0]
			hop.Host = fields[0]
		}
		for _, f := range fields[1:] {
			if v, err := strconv.ParseFloat(strings.TrimSuffix(f, "ms"), 64); err == nil && strings.HasSuffix(f, "ms") {
				switch {
				case hop.RTT1 < 0:
					hop.RTT1 = v
				case hop.RTT2 < 0:
					hop.RTT2 = v
				case hop.RTT3 < 0:
					hop.RTT3 = v
				}
				break
			}
		}
	}
	for _, num := range order {
		r.Hops = append(r.Hops, *hops[num])
	}
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
