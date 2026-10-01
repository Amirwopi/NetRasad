//go:build windows

package diagnostics

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/netrasad/netrasad/internal/platform/executil"
)

func getDefaultGateway() (*GatewayInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "route", "print")
	executil.HideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("route print failed: %w", err)
	}

	return parseWindowsRoute(string(out))
}

var windowsRouteRe = regexp.MustCompile(`^\s*0\.0\.0\.0\s+0\.0\.0\.0\s+(\S+)\s+(\S+)\s+(\d+)\s*$`)

func parseWindowsRoute(output string) (*GatewayInfo, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		m := windowsRouteRe.FindStringSubmatch(line)
		if len(m) >= 4 {
			metric, _ := strconv.Atoi(m[3])
			return &GatewayInfo{
				GatewayIP: m[1],
				Interface: m[2],
				Metric:    metric,
			}, nil
		}
	}
	return nil, fmt.Errorf("default gateway not found in route output")
}
