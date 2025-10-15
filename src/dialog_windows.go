//go:build windows
// +build windows

package main

import (
	"os/exec"
	"strings"
	"syscall"
)

func openModalFolderDialog() (string, bool) {
	// Используем PowerShell с нативным диалогом Windows (полностью скрытый процесс)
	script := `
		Add-Type -AssemblyName System.Windows.Forms
		$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
		$dialog.Description = "Select folder to save profiling results"
		$dialog.ShowNewFolderButton = $true
		$result = $dialog.ShowDialog()
		if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
			Write-Output $dialog.SelectedPath
		}
	`
	
	cmd := exec.Command("powershell", "-WindowStyle", "Hidden", "-Command", script)
	
	// Полностью скрываем окно процесса
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	
	output, err := cmd.Output()
	
	if err != nil {
		return "", false
	}
	
	folderPath := strings.TrimSpace(string(output))
	return folderPath, folderPath != ""
}