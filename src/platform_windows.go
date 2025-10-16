//go:build windows

package main

import (
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
	return filepath.Join("media", "logo_50.png")
}