//go:build !windows

package taskbar

type TopProcessEntry struct {
	Name        string
	DownloadBps float64
	UploadBps   float64
}

type TaskbarOverlayConfig struct {
	DownColorHex  string `json:"downColorHex"`
	UpColorHex    string `json:"upColorHex"`
	BgColorHex    string `json:"bgColorHex"`
	TransparentBg bool   `json:"transparentBg"`
	Position      string `json:"position"`
	Width         int32  `json:"width"`
}

type TaskbarOverlay struct{}

func GetOverlay() *TaskbarOverlay {
	return &TaskbarOverlay{}
}

func (o *TaskbarOverlay) Start() error {
	return nil
}

func (o *TaskbarOverlay) Stop() {}

func (o *TaskbarOverlay) SetEnabled(enabled bool) {}

func (o *TaskbarOverlay) IsEnabled() bool {
	return false
}

func (o *TaskbarOverlay) SetConfig(cfg TaskbarOverlayConfig) {}

func (o *TaskbarOverlay) GetConfig() TaskbarOverlayConfig {
	return TaskbarOverlayConfig{}
}

func (o *TaskbarOverlay) UpdateTopProcesses(list []TopProcessEntry) {}

func (o *TaskbarOverlay) UpdateRates(downBps, upBps float64) {}
