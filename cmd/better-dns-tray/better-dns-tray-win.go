// +build windows

package main

import (
	"os/exec"
	"syscall"
)

func cmdSettings(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
