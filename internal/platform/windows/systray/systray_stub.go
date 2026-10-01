//go:build !windows

package systray

type SysTray struct{}

func GetSysTray() *SysTray {
	return &SysTray{}
}

func (t *SysTray) Start(onOpen, onToggleBand, onExit func()) error {
	return nil
}

func (t *SysTray) Stop() {}
