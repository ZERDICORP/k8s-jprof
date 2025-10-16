//go:build !windows

package main

import (
	"os"
	"path/filepath"

	"gioui.org/font"
	"gioui.org/unit"
)

// TitleFontSize размер шрифта заголовка для macOS и других платформ
const TitleFontSize = unit.Sp(26)

// TitleFont шрифт заголовка для macOS и других платформ
var TitleFont = font.Font{
	Typeface: "Segoe UI",
	Weight:   font.Bold,
}

// getLogoPath возвращает путь к логотипу для macOS и других платформ
func getLogoPath() string {
	exePath, _ := os.Executable() // путь до текущего бинарника
    exeDir := filepath.Dir(exePath)
    return filepath.Join(exeDir, "logo_100.png")
}