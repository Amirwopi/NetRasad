//go:build linux

package platform

import (
	ltraffic "github.com/netrasad/netrasad/internal/platform/linux/traffic"
	"github.com/netrasad/netrasad/internal/ports"
)

func NewTrafficSource() (ports.TrafficSource, error) {
	return ltraffic.NewLinuxSource(), nil
}
