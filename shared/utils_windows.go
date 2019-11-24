// +build windows

package shared

import (
	"os/exec"
	"syscall"
)

func CmdSettings(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
