//go:build linux

package diagnostics

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func getDefaultGateway() (*GatewayInfo, error) {
	gw, err := parseProcNetRoute()
	if err == nil {
		return gw, nil
	}

	return parseIPRoute()
}

func parseProcNetRoute() (*GatewayInfo, error) {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return nil, fmt.Errorf("read /proc/net/route: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("no routes in /proc/net/route")
	}

	for i, line := range lines {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}
		if fields[1] != "00000000" || fields[7] != "00000000" {
			continue
		}

		gatewayHex := fields[2]
		gwIP := parseHexLittleEndianIP(gatewayHex)
		if gwIP == "" {
			continue
		}

		metric := 0
		if len(fields) >= 6 {
			metric, _ = strconv.Atoi(fields[5])
		}

		return &GatewayInfo{
			GatewayIP: gwIP,
			Interface: fields[0],
			Metric:    metric,
		}, nil
	}

	return nil, fmt.Errorf("default gateway not found in /proc/net/route")
}

func parseHexLittleEndianIP(hex string) string {
	if len(hex) != 8 {
		return ""
	}
	bytes := make([]byte, 4)
	for i := 0; i < 4; i++ {
		b, err := strconv.ParseUint(hex[i*2:i*2+2], 16, 8)
		if err != nil {
			return ""
		}
		bytes[3-i] = byte(b)
	}
	return fmt.Sprintf("%d.%d.%d.%d", bytes[0], bytes[1], bytes[2], bytes[3])
}

func parseIPRoute() (*GatewayInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ip", "route", "show", "default")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ip route failed: %w", err)
	}

	return parseIPRouteOutput(string(out))
}

func parseIPRouteOutput(output string) (*GatewayInfo, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "default") {
			continue
		}

		fields := strings.Fields(line)
		gw := ""
		iface := ""
		metric := 0

		for i := 0; i < len(fields); i++ {
			switch fields[i] {
			case "via":
				if i+1 < len(fields) {
					gw = fields[i+1]
					i++
				}
			case "dev":
				if i+1 < len(fields) {
					iface = fields[i+1]
					i++
				}
			case "metric":
				if i+1 < len(fields) {
					metric, _ = strconv.Atoi(fields[i+1])
					i++
				}
			}
		}

		if gw != "" {
			return &GatewayInfo{
				GatewayIP: gw,
				Interface: iface,
				Metric:    metric,
			}, nil
		}
	}

	return nil, fmt.Errorf("default gateway not found in ip route output")
}
