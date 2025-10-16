//go:build windows

package main

import (
	"os"
	"path/filepath"

	"gioui.org/font"
	"gioui.org/unit"
)

// TitleFontSize размер шрифта заголовка для Windows
const TitleFontSize = unit.Sp(26)

// TitleFontFamily название шрифта заголовка для Windows
var TitleFont = font.Font{
	Typeface: "Segoe UI",
	Weight:   font.Bold,
}

// getLogoPath возвращает путь к логотипу для Windows
func getLogoPath() string {
	exePath, _ := os.Executable() // путь до текущего бинарника
    exeDir := filepath.Dir(exePath)
    return filepath.Join(exeDir, "logo_50.png")
}