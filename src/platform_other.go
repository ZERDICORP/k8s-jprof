//go:build !windows

package main

import (
	"path/filepath"

	"gioui.org/unit"
)

// TitleFontSize размер шрифта заголовка для macOS и других платформ
const TitleFontSize = unit.Sp(26)

// TitleVerticalOffset вертикальная корректировка для выравнивания с логотипом на macOS
const TitleVerticalOffset = unit.Dp(0)

// getLogoPath возвращает путь к логотипу для macOS и других платформ
func getLogoPath() string {
	return filepath.Join("media", "logo_100.png")
}