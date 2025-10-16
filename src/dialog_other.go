//go:build !windows
// +build !windows

package main

import (
	"os/exec"
	"runtime"
	"strings"
)

var dialogProcess *exec.Cmd

func openModalFolderDialog() (string, bool) {
	var cmd *exec.Cmd
	
	if runtime.GOOS == "darwin" {
		// macOS - используем osascript
		cmd = exec.Command("osascript", "-e", `choose folder with prompt "Select folder to save profiling results"`)
	} else {
		// Linux - используем zenity
		cmd = exec.Command("zenity", "--file-selection", "--directory", "--title=Select folder to save profiling results")
	}
	
	// Сохраняем ссылку на процесс для возможности его закрытия
	dialogProcess = cmd
	
	output, err := cmd.Output()
	
	// Очищаем ссылку после завершения
	dialogProcess = nil
	
	if err != nil {
		return "", false
	}
	
	folder := strings.TrimSpace(string(output))
	
	// На macOS osascript возвращает path в формате "alias:Macintosh HD:Users:..."
	if runtime.GOOS == "darwin" && strings.HasPrefix(folder, "alias") {
		// Конвертируем alias в обычный путь
		convertCmd := exec.Command("osascript", "-e", `tell application "Finder" to return POSIX path of (choose folder with prompt "Select folder to save profiling results")`)
		convertOutput, convertErr := convertCmd.Output()
		if convertErr == nil {
			folder = strings.TrimSpace(string(convertOutput))
		}
	}
	
	return folder, folder != ""
}

func closeAnyOpenDialogs() {
	if dialogProcess != nil && dialogProcess.Process != nil {
		dialogProcess.Process.Kill()
		dialogProcess = nil
	}
}