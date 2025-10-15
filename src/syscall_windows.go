//go:build windows
// +build windows

package main

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr устанавливает атрибуты процесса для Windows
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}