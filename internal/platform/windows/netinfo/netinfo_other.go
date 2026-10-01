//go:build !windows

package netinfo

type AdapterInfo struct {
	IPAddresses []string
	GatewayIP string
	SSID string
}

func GetAdapterInfo() map[uint64]AdapterInfo {
	return map[uint64]AdapterInfo{}
}

func GetWiFiSSID() string {
	return ""
}
