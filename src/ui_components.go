//go:build windows
// +build windows

package main

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// DropdownItem представляет элемент в выпадающем списке
type DropdownItem struct {
	Text      string
	Selected  bool
	Clickable *widget.Clickable
}

// DrawDropdownWithBorder draws dropdown list with border and background
func DrawDropdownWithBorder(gtx layout.Context, th *material.Theme, items []DropdownItem, onItemClick func(int)) layout.Dimensions {
	return DrawDropdownWithList(gtx, th, items, onItemClick, &widget.List{
		List: layout.List{
			Axis: layout.Vertical,
		},
	})
}

// DrawDropdownWithList draws dropdown list with border, background and custom list widget
func DrawDropdownWithList(gtx layout.Context, th *material.Theme, items []DropdownItem, onItemClick func(int), list *widget.List) layout.Dimensions {
	if len(items) == 0 {
		return layout.Dimensions{}
	}

	// Ограничиваем высоту списка
	maxHeight := gtx.Dp(unit.Dp(150))
	if gtx.Constraints.Max.Y < maxHeight {
		maxHeight = gtx.Constraints.Max.Y
	}
	gtx.Constraints.Max.Y = maxHeight

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			// Фон dropdown с бордером
			defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, color.NRGBA{R: 220, G: 220, B: 220, A: 255}) // Серый бордер
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(1), Right: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				// Белый фон внутри
				defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
				paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

				return material.List(th, list).Layout(gtx, len(items), func(gtx layout.Context, index int) layout.Dimensions {
					item := items[index]

					// Обработка клика
					if item.Clickable.Clicked(gtx) {
						onItemClick(index)
					}

					return material.Clickable(gtx, item.Clickable, func(gtx layout.Context) layout.Dimensions {
						// Фон для выбранного элемента
						if item.Selected {
							defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
							paint.Fill(gtx.Ops, color.NRGBA{R: 230, G: 230, B: 230, A: 255})
						}

						// Используем Flexed(1) чтобы элемент занял всю ширину
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									label := material.Label(th, unit.Sp(14), item.Text)
									if item.Selected {
										label.Color = color.NRGBA{R: 30, G: 30, B: 30, A: 255}
									} else {
										label.Color = color.NRGBA{R: 60, G: 60, B: 60, A: 255}
									}
									return label.Layout(gtx)
								})
							}),
						)
					})
				})
			})
		},
	)
}

// DrawSearchField рисует поле поиска с фоном
func DrawSearchField(gtx layout.Context, th *material.Theme, editor *widget.Editor, placeholder string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			// Фон для поля поиска
			defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, color.NRGBA{R: 248, G: 248, B: 248, A: 255})

			editorWidget := material.Editor(th, editor, placeholder)
			editorWidget.Editor.SingleLine = true
			editorWidget.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
			editorWidget.HintColor = color.NRGBA{R: 120, G: 120, B: 120, A: 255}

			return editorWidget.Layout(gtx)
		})
	})
}

// DrawDropdownContainer рисует контейнер для dropdown с border
func DrawDropdownContainer(gtx layout.Context, th *material.Theme, content func(gtx layout.Context) layout.Dimensions) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			// Серый бордер
			defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, color.NRGBA{R: 180, G: 180, B: 180, A: 255})
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
		func(gtx layout.Context) layout.Dimensions {
			// Внутренний контент с отступом для бордера
			return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(1), Right: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				// Белый фон внутри
				defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
				paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

				return content(gtx)
			})
		},
	)
}
