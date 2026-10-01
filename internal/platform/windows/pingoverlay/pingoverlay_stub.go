//go:build !windows

package pingoverlay

type Config struct {
	Enabled bool   `json:"enabled"`
	Address string `json:"address"`
	Label   string `json:"label"`
	X       int32  `json:"x"`
	Y       int32  `json:"y"`
}

type PingOverlay struct{}

var globalOverlay = &PingOverlay{}

func GetOverlay() *PingOverlay {
	return globalOverlay
}

func (o *PingOverlay) SetConfig(cfg Config) {}

func (o *PingOverlay) GetConfig() Config {
	return Config{}
}
