//go:build windows

package main

import (
	"path/filepath"

	"gioui.org/unit"
)

// TitleFontSize размер шрифта заголовка для Windows
const TitleFontSize = unit.Sp(26)

// getLogoPath возвращает путь к логотипу для Windows
func getLogoPath() string {
	return filepath.Join("media", "logo_50.png")
}