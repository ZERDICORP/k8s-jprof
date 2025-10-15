package main

import (
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// TitleComponent отрисовывает заголовок с логотипом
type TitleComponent struct {
	logoImage paint.ImageOp
}

func NewTitleComponent(logoImage paint.ImageOp) *TitleComponent {
	return &TitleComponent{
		logoImage: logoImage,
	}
}

// Layout отрисовывает логотип и заголовок слева
func (tc *TitleComponent) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		// Логотип
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if tc.logoImage.Size().X > 0 {
				// Отрисовываем логотип как есть (уже 24x24)
				tc.logoImage.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				return layout.Dimensions{Size: tc.logoImage.Size()}
			}
			return layout.Dimensions{}
		}),
		// Отступ между логотипом и текстом
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
		}),
		// Текст заголовка
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Label(th, unit.Sp(26), "k8s-pfr.beta")
			label.Font.Weight = font.ExtraBold
			label.Color = th.Palette.Fg
			return label.Layout(gtx)
		}),
	)
}

// CenteredTitleComponent отрисовывает заголовок по центру (для режима инициализации)
func (tc *TitleComponent) CenteredLayout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// Логотип
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if tc.logoImage.Size().X > 0 {
						tc.logoImage.Add(gtx.Ops)
						paint.PaintOp{}.Add(gtx.Ops)
						return layout.Dimensions{Size: tc.logoImage.Size()}
					}
					return layout.Dimensions{}
				}),
				// Отступ
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Width: unit.Dp(12)}.Layout(gtx)
				}),
				// Текст заголовка
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := material.Label(th, unit.Sp(26), "k8s-pfr.beta")
					label.Font.Weight = font.ExtraBold
					label.Color = th.Palette.Fg
					return label.Layout(gtx)
				}),
			)
		}),
	)
}

// VersionBadgeComponent отрисовывает версию async-profiler справа
type VersionBadgeComponent struct {
	versionBadge *widget.Clickable
	version      string
}

func NewVersionBadgeComponent(versionBadge *widget.Clickable, version string) *VersionBadgeComponent {
	return &VersionBadgeComponent{
		versionBadge: versionBadge,
		version:      version,
	}
}

func (vbc *VersionBadgeComponent) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	badge := material.Button(th, vbc.versionBadge, "async-profiler "+vbc.version)
	badge.Background = th.Palette.ContrastBg
	badge.Color = th.Palette.ContrastFg
	badge.TextSize = unit.Sp(12)
	return badge.Layout(gtx)
}

// HeaderComponent объединяет заголовок и версию
type HeaderComponent struct {
	titleComponent       *TitleComponent
	versionBadgeComponent *VersionBadgeComponent
}

func NewHeaderComponent(logoImage paint.ImageOp, versionBadge *widget.Clickable, version string) *HeaderComponent {
	return &HeaderComponent{
		titleComponent:        NewTitleComponent(logoImage),
		versionBadgeComponent: NewVersionBadgeComponent(versionBadge, version),
	}
}

// Layout отрисовывает полный заголовок с версией справа
func (hc *HeaderComponent) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(16), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// Логотип и название слева
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return hc.titleComponent.Layout(gtx, th)
			}),
			// Пространство между логотипом/названием и версией
			layout.Flexed(1, layout.Spacer{}.Layout),
			// Версия async-profiler справа
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return hc.versionBadgeComponent.Layout(gtx, th)
			}),
		)
	})
}

// SimpleHeaderLayout отрисовывает простой заголовок без версии (для режимов ошибки/инициализации)
func (hc *HeaderComponent) SimpleHeaderLayout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(50)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return hc.titleComponent.Layout(gtx, th)
	})
}