//go:build windows

package main

import (
	"path/filepath"

	"gioui.org/unit"
)

// TitleFontSize размер шрифта заголовка для Windows
const TitleFontSize = unit.Sp(26)

// TitleVerticalOffset вертикальная корректировка для выравнивания с логотипом на Windows
const TitleVerticalOffset = unit.Dp(0)

// getLogoPath возвращает путь к логотипу для Windows
func getLogoPath() string {
	return filepath.Join("media", "logo_50.png")
}