//go:build !windows
// +build !windows

package main

import "os/exec"

// setSysProcAttr для non-Windows платформ - ничего не делает
func setSysProcAttr(cmd *exec.Cmd) {
	// На Unix-подобных системах нет необходимости скрывать окна
}