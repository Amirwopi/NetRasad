//go:build windows

package platform

import (
	wtraffic "github.com/netrasad/netrasad/internal/platform/windows/traffic"
	"github.com/netrasad/netrasad/internal/ports"
)

func NewTrafficSource() (ports.TrafficSource, error) {
	return wtraffic.NewWindowsSource(), nil
}
