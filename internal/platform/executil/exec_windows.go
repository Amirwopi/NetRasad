//go:build windows

package executil

import (
	"os/exec"
	"syscall"
)

const CREATE_NO_WINDOW = 0x08000000

func HideWindow(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= CREATE_NO_WINDOW
}
