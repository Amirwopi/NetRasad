//go:build linux

package diagnostics

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/netrasad/netrasad/internal/platform/linux/netinfo"
)

func getDefaultGateway() (*GatewayInfo, error) {
	routes, err := netinfo.DefaultRoutes()
	if err == nil && len(routes) > 0 {
		r := routes[0]
		return &GatewayInfo{
			GatewayIP: r.Gateway,
			Interface: r.Interface,
			Metric:    r.Metric,
		}, nil
	}

	return parseIPRoute()
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
