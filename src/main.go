package main

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Function to open folder selection dialog
func chooseFolderDialog() (string, error) {
	// Используем нативный модальный диалог
	folder, ok := openModalFolderDialog()
	if !ok {
		return "", fmt.Errorf("user cancelled selection")
	}
	return folder, nil
}

type KubeconfigSelector struct {
	configs         []string
	filteredConfigs []string
	selectedConfig  string
	expanded        bool
	button          widget.Clickable
	list            widget.List
	clickables      []widget.Clickable
	searchEditor    widget.Editor
	searchText      string
	loading         bool
}

type FormatSelector struct {
	formats         []string
	filteredFormats []string
	selectedFormat  string
	expanded        bool
	button          widget.Clickable
	list            widget.List
	clickables      []widget.Clickable
	searchEditor    widget.Editor
	searchText      string
	loading         bool
}

func NewKubeconfigSelector() *KubeconfigSelector {
	ks := &KubeconfigSelector{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		searchEditor: widget.Editor{
			SingleLine: true,
		},
	}
	ks.scanKubeconfigs()
	ks.loadSelection()
	return ks
}

func (ks *KubeconfigSelector) scanKubeconfigs() {
	ks.configs = []string{}

	kubeDir := getKubeDir()
	if _, err := os.Stat(kubeDir); os.IsNotExist(err) {
		return
	}

	entries, err := os.ReadDir(kubeDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			ks.configs = append(ks.configs, entry.Name())
		}
	}

	ks.filteredConfigs = make([]string, len(ks.configs))
	copy(ks.filteredConfigs, ks.configs)
	ks.clickables = make([]widget.Clickable, len(ks.filteredConfigs))
}

func (ks *KubeconfigSelector) filterConfigs() {
	if ks.searchText == "" {
		ks.filteredConfigs = make([]string, len(ks.configs))
		copy(ks.filteredConfigs, ks.configs)
	} else {
		ks.filteredConfigs = []string{}
		searchLower := strings.ToLower(ks.searchText)
		for _, config := range ks.configs {
			if strings.Contains(strings.ToLower(config), searchLower) {
				ks.filteredConfigs = append(ks.filteredConfigs, config)
			}
		}
		// Если ничего не найдено, добавляем "(нет)"
		if len(ks.filteredConfigs) == 0 {
			ks.filteredConfigs = []string{"(нет)"}
		}
	}
	ks.clickables = make([]widget.Clickable, len(ks.filteredConfigs))
}

func (ks *KubeconfigSelector) loadSelection() {
	configFile := getConfigFilePath()
	data, err := os.ReadFile(configFile)
	if err == nil {
		saved := strings.TrimSpace(string(data))
		// Check that the saved file still exists
		for _, config := range ks.configs {
			if config == saved {
				ks.selectedConfig = saved
				return
			}
		}
	}
	// If no saved selection or file not found - leave empty
	ks.selectedConfig = ""
}

func (ks *KubeconfigSelector) saveSelection() {
	if ks.selectedConfig == "" {
		return
	}
	configFile := getConfigFilePath()
	os.MkdirAll(filepath.Dir(configFile), 0755)
	os.WriteFile(configFile, []byte(ks.selectedConfig), 0644)
}

func getConfigFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".k8s-pfr", "kubeconfig.mem")
}

func getKubeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".kube")
}

func (ks *KubeconfigSelector) Layout(gtx layout.Context, th *material.Theme, app *Application) layout.Dimensions {
	// Update search text
	if ks.searchEditor.Text() != ks.searchText {
		ks.searchText = ks.searchEditor.Text()
		ks.filterConfigs()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Fixed width for label - 120dp
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
					return material.Label(th, unit.Sp(16), "Kubeconfig:").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					// Handle button click
					for ks.button.Clicked(gtx) {
						// Если этот селект уже открыт - просто закрываем его
						if ks.expanded {
							ks.expanded = false
						} else {
							// Если закрыт - закрываем все остальные и открываем этот
							app.closeAllSelectors()
							ks.expanded = true
							// Clear search when opening
							ks.searchEditor.SetText("")
							ks.searchText = ""
							ks.filterConfigs()
						}
					}

					// Pointer cursor on hover
					if ks.button.Hovered() {
						pointer.CursorPointer.Add(gtx.Ops)
					}

					// Button text
					buttonText := "Select kubeconfig file"
					if ks.selectedConfig != "" {
						buttonText = ks.selectedConfig
					} else if len(ks.configs) == 0 {
						buttonText = "No files available"
					}

					btn := material.Button(th, &ks.button, buttonText)
					if ks.selectedConfig == "" && len(ks.configs) > 0 {
						// Gray color for inactive button
						btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						btn.Color = color.NRGBA{R: 80, G: 80, B: 80, A: 255}
					} else {
						// Normal colors for active button
						btn.Background = color.NRGBA{R: 240, G: 240, B: 240, A: 255}
						btn.Color = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
					}
					return btn.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !ks.expanded {
				return layout.Dimensions{}
			}

			// Draw border around container
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					// Gray border
					defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
					paint.Fill(gtx.Ops, color.NRGBA{R: 180, G: 180, B: 180, A: 255})
					return layout.Dimensions{Size: gtx.Constraints.Max}
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(1), Right: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// White background inside
						defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							// Search field
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									// Фон для поля поиска на всю ширину
									defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
									paint.Fill(gtx.Ops, color.NRGBA{R: 248, G: 248, B: 248, A: 255})

									return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										editor := material.Editor(th, &ks.searchEditor, "Search kubeconfig...")
										editor.Editor.SingleLine = true
										editor.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
										editor.HintColor = color.NRGBA{R: 120, G: 120, B: 120, A: 255}

										return editor.Layout(gtx)
									})
								})
							}),
							// Список конфигов - занимает все доступное место
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								if len(ks.filteredConfigs) == 0 {
									// Если нет результатов поиска, показываем пустое пространство
									return layout.Dimensions{}
								}
								return material.List(th, &ks.list).Layout(gtx, len(ks.filteredConfigs), func(gtx layout.Context, index int) layout.Dimensions {
									if index >= len(ks.clickables) {
										return layout.Dimensions{}
									}

									// Обработка клика по элементу списка
									for ks.clickables[index].Clicked(gtx) {
										newConfig := ks.filteredConfigs[index]
										
										// Если выбрали "(нет)" - очищаем выбор
										if newConfig == "(нет)" {
											ks.selectedConfig = ""
											ks.expanded = false
											ks.saveSelection()
											
											// Сбрасываем namespace и pod
											app.namespaceSelector.selectedNamespace = ""
											app.namespaceSelector.namespaces = []string{}
											app.podSelector.selectedPod = ""
											app.podSelector.pods = []string{}
											
											// Очищаем статус записи и кнопку браузера
											app.recordingResult = ""
											app.showBrowserButton = false
											app.htmlOutputPath = ""
											app.hasCompletedRecording = false
											break
										}
										
										// Если выбираем тот же конфиг - просто закрываем селект
										if newConfig == ks.selectedConfig {
											ks.expanded = false
											break
										}
										
										// Меняем конфиг - сбрасываем namespace и pod
										ks.selectedConfig = newConfig
										ks.expanded = false
										ks.saveSelection()

										// Сбрасываем namespace и pod при смене конфига
										app.namespaceSelector.selectedNamespace = ""
										app.namespaceSelector.namespaces = []string{}
										app.podSelector.selectedPod = ""
										app.podSelector.pods = []string{}

										// Очищаем статус записи и кнопку браузера
										app.recordingResult = ""
										app.showBrowserButton = false
										app.htmlOutputPath = ""
										app.hasCompletedRecording = false

										// Show loading screen when changing kubeconfig
										app.isLoading = true
										app.loadingStartTime = time.Now()

										// Загружаем namespaces для нового конфига
										go func() {
											// Небольшая задержка для UI
											time.Sleep(200 * time.Millisecond)
											app.namespaceSelector.LoadNamespaces(ks.selectedConfig, app)
										}()
									}

									// Стиль элемента
									isSelected := ks.filteredConfigs[index] == ks.selectedConfig

									return material.Clickable(gtx, &ks.clickables[index], func(gtx layout.Context) layout.Dimensions {
										// Фон для выбранного элемента
										if isSelected {
											defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
											paint.Fill(gtx.Ops, color.NRGBA{R: 220, G: 220, B: 220, A: 255})
										}

										// Pointer cursor при наведении
										if ks.clickables[index].Hovered() {
											pointer.CursorPointer.Add(gtx.Ops)
										}

										// Используем Flexed(1) чтобы элемент занял всю ширину
										return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
											layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													label := material.Label(th, unit.Sp(14), ks.filteredConfigs[index])
													if isSelected {
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
							}),
						)
					})
				},
			)
		}),
	)
}

func (ks *KubeconfigSelector) GetSelectedConfig() string {
	return ks.selectedConfig
}

func (ks *KubeconfigSelector) IsConfigSelected() bool {
	return ks.selectedConfig != ""
}

func (ks *KubeconfigSelector) IsExpanded() bool {
	return ks.expanded
}

type NamespaceSelector struct {
	namespaces         []string
	filteredNamespaces []string
	selectedNamespace  string
	expanded           bool
	button             widget.Clickable
	list               widget.List
	clickables         []widget.Clickable
	searchEditor       widget.Editor
	searchText         string
	loading            bool
}

func NewNamespaceSelector() *NamespaceSelector {
	ns := &NamespaceSelector{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		searchEditor: widget.Editor{
			SingleLine: true,
		},
	}
	ns.loadSelection()
	return ns
}

func (ns *NamespaceSelector) LoadNamespaces(kubeconfigPath string, app *Application) {
	if kubeconfigPath == "" {
		ns.namespaces = []string{}
		ns.filteredNamespaces = []string{}
		ns.clickables = []widget.Clickable{}
		return
	}

	ns.loading = true
	app.isLoading = true
	app.loadingStartTime = time.Now()

	// Asynchronous namespace loading
	go func() {
		namespaces := ns.getNamespacesFromKubeconfig(kubeconfigPath)

		// Проверяем на ошибки сети (если не удалось получить namespaces)
		if len(namespaces) == 0 {
			// Проверяем, что это действительно ошибка сети, а не пустой результат
			// Попробуем простую команду kubectl version для проверки доступности кластера
			cmd := exec.Command("kubectl", "--kubeconfig", filepath.Join(getKubeDir(), kubeconfigPath), "version", "--short")
			setSysProcAttr(cmd)
			if err := cmd.Run(); err != nil {
				// Если kubectl version не работает, значит проблема с сетью/кластером
				app.hasNetworkError = true
				app.lastFailedAction = "loadNamespaces"
				ns.loading = false
				app.isLoading = false
				if app.invalidate != nil {
					app.invalidate()
				}
				return
			}
		}

		// Обновляем в UI потоке
		ns.namespaces = namespaces
		ns.filteredNamespaces = make([]string, len(namespaces))
		copy(ns.filteredNamespaces, namespaces)
		ns.clickables = make([]widget.Clickable, len(namespaces))

		// Проверяем, что сохраненный namespace все еще существует
		saved := ns.selectedNamespace
		ns.selectedNamespace = ""
		for _, ns_name := range namespaces {
			if ns_name == saved {
				ns.selectedNamespace = saved
				break
			}
		}

		ns.loading = false
		app.isLoading = false
		if app.invalidate != nil {
			app.invalidate()
		}
	}()
}

func (ns *NamespaceSelector) getNamespacesFromKubeconfig(kubeconfigPath string) []string {
	if kubeconfigPath == "" {
		return []string{}
	}

	fullPath := filepath.Join(getKubeDir(), kubeconfigPath)

	cmd := exec.Command("kubectl", "--kubeconfig", fullPath, "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}")

	// Устанавливаем атрибуты процесса для скрытия окна терминала (Windows)
	setSysProcAttr(cmd)

	// Захватываем stderr для диагностики
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		errMsg := fmt.Sprintf("Ошибка получения namespaces: %v", err)
		if stderr.Len() > 0 {
			errMsg += fmt.Sprintf(" | Details: %s", stderr.String())
		}
		log.Print(errMsg)
		return []string{}
	}

	namespaces := strings.Fields(string(output))
	sort.Strings(namespaces)
	return namespaces
}

func (ns *NamespaceSelector) filterNamespaces() {
	if ns.searchText == "" {
		ns.filteredNamespaces = make([]string, len(ns.namespaces))
		copy(ns.filteredNamespaces, ns.namespaces)
	} else {
		ns.filteredNamespaces = []string{}
		searchLower := strings.ToLower(ns.searchText)
		for _, namespace := range ns.namespaces {
			if strings.Contains(strings.ToLower(namespace), searchLower) {
				ns.filteredNamespaces = append(ns.filteredNamespaces, namespace)
			}
		}
		// Если ничего не найдено, добавляем "(нет)"
		if len(ns.filteredNamespaces) == 0 {
			ns.filteredNamespaces = []string{"(нет)"}
		}
	}
	ns.clickables = make([]widget.Clickable, len(ns.filteredNamespaces))
}

func (ns *NamespaceSelector) loadSelection() {
	configFile := getNamespaceConfigFilePath()
	data, err := os.ReadFile(configFile)
	if err == nil {
		ns.selectedNamespace = strings.TrimSpace(string(data))
	}
}

func (ns *NamespaceSelector) saveSelection() {
	if ns.selectedNamespace == "" {
		return
	}
	configFile := getNamespaceConfigFilePath()
	os.MkdirAll(filepath.Dir(configFile), 0755)
	os.WriteFile(configFile, []byte(ns.selectedNamespace), 0644)
}

func getNamespaceConfigFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".k8s-pfr", "namespace.mem")
}

func (ns *NamespaceSelector) Layout(gtx layout.Context, th *material.Theme, app *Application) layout.Dimensions {
	// Update search text
	if ns.searchEditor.Text() != ns.searchText {
		ns.searchText = ns.searchEditor.Text()
		ns.filterNamespaces()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Fixed width for label - 120dp
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
					return material.Label(th, unit.Sp(16), "Namespace:").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					// Handle button click
					for ns.button.Clicked(gtx) {
						// Если этот селект уже открыт - просто закрываем его
						if ns.expanded {
							ns.expanded = false
						} else {
							// Если закрыт - закрываем все остальные и открываем этот
							app.closeAllSelectors()
							ns.expanded = true
							// Clear search when opening
							ns.searchEditor.SetText("")
							ns.searchText = ""
							ns.filterNamespaces()
						}
					}

					// Pointer cursor on hover
					if ns.button.Hovered() {
						pointer.CursorPointer.Add(gtx.Ops)
					}

					// Button text
					buttonText := "Select namespace"
					if ns.loading {
						buttonText = "Loading..."
					} else if ns.selectedNamespace != "" {
						buttonText = ns.selectedNamespace
					} else if len(ns.namespaces) == 0 {
						buttonText = "No namespaces available"
					}

					btn := material.Button(th, &ns.button, buttonText)
					if ns.selectedNamespace == "" && len(ns.namespaces) > 0 {
						btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						btn.Color = color.NRGBA{R: 80, G: 80, B: 80, A: 255}
					} else {
						btn.Background = color.NRGBA{R: 240, G: 240, B: 240, A: 255}
						btn.Color = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
					}
					return btn.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !ns.expanded {
				return layout.Dimensions{}
			}

			// Draw border around container
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					// Gray border
					defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
					paint.Fill(gtx.Ops, color.NRGBA{R: 180, G: 180, B: 180, A: 255})
					return layout.Dimensions{Size: gtx.Constraints.Max}
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(1), Right: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// White background inside
						defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				// Поле поиска
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// Фон для поля поиска на всю ширину
						defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, color.NRGBA{R: 248, G: 248, B: 248, A: 255})

						return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							editor := material.Editor(th, &ns.searchEditor, "Search namespace...")
							editor.Editor.SingleLine = true
							editor.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
							editor.HintColor = color.NRGBA{R: 120, G: 120, B: 120, A: 255}

							return editor.Layout(gtx)
						})
					})
				}),
				// Список namespaces - занимает все доступное место
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					if len(ns.filteredNamespaces) == 0 {
						// Если нет результатов поиска, показываем пустое пространство
						return layout.Dimensions{}
					}
					return material.List(th, &ns.list).Layout(gtx, len(ns.filteredNamespaces), func(gtx layout.Context, index int) layout.Dimensions {
						if index >= len(ns.clickables) {
							return layout.Dimensions{}
						}

						// Обработка клика по элементу списка
						for ns.clickables[index].Clicked(gtx) {
							newNamespace := ns.filteredNamespaces[index]
							
							// Если кликнули по "(нет)" - очищаем выбор
							if newNamespace == "(нет)" {
								ns.selectedNamespace = ""
								ns.expanded = false
								ns.saveSelection()
								
								// Сбрасываем pod при очистке namespace
								app.podSelector.selectedPod = ""
								app.podSelector.pods = []string{}
								
								// Очищаем статус записи и кнопку браузера
								app.recordingResult = ""
								app.showBrowserButton = false
								app.htmlOutputPath = ""
								app.hasCompletedRecording = false
								break
							}
							
							// Если выбираем тот же namespace - просто закрываем селект
							if newNamespace == ns.selectedNamespace {
								ns.expanded = false
								break
							}
							
							// Меняем namespace - сбрасываем pod
							ns.selectedNamespace = newNamespace
							ns.expanded = false
							ns.saveSelection()

							// Сбрасываем pod при смене namespace
							app.podSelector.selectedPod = ""
							app.podSelector.pods = []string{}

							// Очищаем статус записи и кнопку браузера
							app.recordingResult = ""
							app.showBrowserButton = false
							app.htmlOutputPath = ""
							app.hasCompletedRecording = false

							// Загружаем поды для выбранного namespace
							selectedConfig := app.kubeconfigSelector.GetSelectedConfig()
							if selectedConfig != "" {
								app.podSelector.LoadPods(selectedConfig, ns.selectedNamespace, app)
							}
						}

						// Стиль элемента
						isSelected := ns.filteredNamespaces[index] == ns.selectedNamespace

						return material.Clickable(gtx, &ns.clickables[index], func(gtx layout.Context) layout.Dimensions {
							// Фон для выбранного элемента
							if isSelected {
								defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
								paint.Fill(gtx.Ops, color.NRGBA{R: 220, G: 220, B: 220, A: 255})
							}

							// Pointer cursor при наведении
							if ns.clickables[index].Hovered() {
								pointer.CursorPointer.Add(gtx.Ops)
							}

							// Используем Flexed(1) чтобы элемент занял всю ширину
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										label := material.Label(th, unit.Sp(14), ns.filteredNamespaces[index])
										if isSelected {
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
				}),
			)
					})
				},
			)
		}),
	)
}

func (ns *NamespaceSelector) GetSelectedNamespace() string {
	return ns.selectedNamespace
}

func (ns *NamespaceSelector) IsNamespaceSelected() bool {
	return ns.selectedNamespace != ""
}

func (ns *NamespaceSelector) Reset() {
	ns.namespaces = []string{}
	ns.filteredNamespaces = []string{}
	ns.selectedNamespace = ""
	ns.expanded = false
	ns.searchEditor.SetText("")
	ns.searchText = ""
}

func (ns *NamespaceSelector) IsExpanded() bool {
	return ns.expanded
}

func NewFormatSelector() *FormatSelector {
	fs := &FormatSelector{
		formats: []string{"(none)", "html", "collapsed", "pprof", "pb.gz", "heatmap", "otlp"},
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		searchEditor: widget.Editor{
			SingleLine: true,
		},
	}
	fs.loadSelection()
	fs.filterFormats()
	return fs
}

func (fs *FormatSelector) filterFormats() {
	if fs.searchText == "" {
		fs.filteredFormats = fs.formats
		return
	}

	fs.filteredFormats = []string{}
	searchLower := strings.ToLower(fs.searchText)
	for _, format := range fs.formats {
		if strings.Contains(strings.ToLower(format), searchLower) {
			fs.filteredFormats = append(fs.filteredFormats, format)
		}
	}
	
	// Если после фильтрации ничего не найдено, добавляем "(нет)"
	if len(fs.filteredFormats) == 0 {
		fs.filteredFormats = []string{"(нет)"}
	}
}

func (fs *FormatSelector) loadSelection() {
	configFile := getFormatConfigFilePath()
	data, err := os.ReadFile(configFile)
	if err == nil {
		saved := strings.TrimSpace(string(data))
		// Check if format is available
		for _, format := range fs.formats {
			if format == saved {
				fs.selectedFormat = saved
				return
			}
		}
	}
	// Default select "heatmap" и сохраняем это значение
	fs.selectedFormat = "heatmap"
	os.MkdirAll(filepath.Dir(configFile), 0755)
	os.WriteFile(configFile, []byte("heatmap"), 0644)
}

func (fs *FormatSelector) saveSelection() {
	configFile := getFormatConfigFilePath()
	os.MkdirAll(filepath.Dir(configFile), 0755)
	
	// Если выбран "(нет)" (из поиска) - сохраняем как "(none)"
	valueToSave := fs.selectedFormat
	if valueToSave == "" {
		valueToSave = "(none)"
	}
	
	os.WriteFile(configFile, []byte(valueToSave), 0644)
}

func (fs *FormatSelector) GetSelectedFormat() string {
	return fs.selectedFormat
}

func (fs *FormatSelector) IsExpanded() bool {
	return fs.expanded
}

func (fs *FormatSelector) Layout(gtx layout.Context, th *material.Theme, app *Application) layout.Dimensions {
	// Update search text
	if fs.searchEditor.Text() != fs.searchText {
		fs.searchText = fs.searchEditor.Text()
		fs.filterFormats()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Fixed width for label - 120dp
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
					return material.Label(th, unit.Sp(16), "Convert to:").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					// Handle button click
					for fs.button.Clicked(gtx) {
						// Если этот селект уже открыт - просто закрываем его
						if fs.expanded {
							fs.expanded = false
						} else {
							// Если закрыт - закрываем все остальные и открываем этот
							app.closeAllSelectors()
							fs.expanded = true
							// Clear search when opening
							fs.searchEditor.SetText("")
							fs.searchText = ""
							fs.filterFormats()
						}
					}

					// Pointer cursor on hover
					if fs.button.Hovered() {
						pointer.CursorPointer.Add(gtx.Ops)
					}

					// Button text
					buttonText := "Select format"
					if fs.selectedFormat != "" {
						buttonText = fs.selectedFormat
					} else if len(fs.formats) == 0 {
						buttonText = "No formats available"
					}

					btn := material.Button(th, &fs.button, buttonText)
					if fs.selectedFormat == "" && len(fs.formats) > 0 {
						// Gray color for inactive button
						btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						btn.Color = color.NRGBA{R: 80, G: 80, B: 80, A: 255}
					} else {
						// Normal colors for active button
						btn.Background = color.NRGBA{R: 240, G: 240, B: 240, A: 255}
						btn.Color = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
					}
					return btn.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !fs.expanded || len(fs.filteredFormats) == 0 {
				return layout.Dimensions{}
			}

			// Draw border around container
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					// Gray border
					defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
					paint.Fill(gtx.Ops, color.NRGBA{R: 180, G: 180, B: 180, A: 255})
					return layout.Dimensions{Size: gtx.Constraints.Max}
				},

				func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(1), Right: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// White background inside
						defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							// Search field
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									// Фон для поля поиска на всю ширину
									defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
									paint.Fill(gtx.Ops, color.NRGBA{R: 248, G: 248, B: 248, A: 255})

									return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										editor := material.Editor(th, &fs.searchEditor, "Search formats...")
										editor.Editor.SingleLine = true
										editor.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
										editor.HintColor = color.NRGBA{R: 120, G: 120, B: 120, A: 255}

										return editor.Layout(gtx)
									})
								})
							}),
							// List of formats
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								// Limit height to show max 5 items
								maxHeight := gtx.Dp(unit.Dp(200))
								if gtx.Constraints.Max.Y > maxHeight {
									gtx.Constraints.Max.Y = maxHeight
								}

								// Initialize clickables if needed
								if len(fs.clickables) < len(fs.filteredFormats) {
									fs.clickables = make([]widget.Clickable, len(fs.filteredFormats))
								}

								return material.List(th, &fs.list).Layout(gtx, len(fs.filteredFormats), func(gtx layout.Context, index int) layout.Dimensions {
									// Handle item click
									for fs.clickables[index].Clicked(gtx) {
										newFormat := fs.filteredFormats[index]
										
										// Если кликнули по "(нет)" - очищаем выбор
										if newFormat == "(нет)" {
											fs.selectedFormat = ""
											fs.expanded = false
											fs.saveSelection()
											if app.invalidate != nil {
												app.invalidate()
											}
											break
										}
										
										// Если выбираем тот же формат - просто закрываем селект
										if newFormat == fs.selectedFormat {
											fs.expanded = false
											break
										}
										
										// Меняем формат
										fs.selectedFormat = newFormat
										fs.expanded = false
										fs.saveSelection()
										if app.invalidate != nil {
											app.invalidate()
										}
									}

									// Item style
									isSelected := fs.filteredFormats[index] == fs.selectedFormat

									return material.Clickable(gtx, &fs.clickables[index], func(gtx layout.Context) layout.Dimensions {
										// Background for selected item
										if isSelected {
											defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
											paint.Fill(gtx.Ops, color.NRGBA{R: 220, G: 220, B: 220, A: 255})
										}

										// Pointer cursor on hover
										if fs.clickables[index].Hovered() {
											pointer.CursorPointer.Add(gtx.Ops)
										}

										return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
											layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
													label := material.Label(th, unit.Sp(14), fs.filteredFormats[index])
													if isSelected {
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
							}),
						)
					})
				},
			)
		}),
	)
}

// PodSelector управляет выбором подов
type PodSelector struct {
	pods         []string
	filteredPods []string
	selectedPod  string
	expanded     bool
	button       widget.Clickable
	list         widget.List
	clickables   []widget.Clickable
	searchEditor widget.Editor
	searchText   string
	loading      bool
}

func NewPodSelector() *PodSelector {
	ps := &PodSelector{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		searchEditor: widget.Editor{
			SingleLine: true,
		},
	}
	ps.loadSelection()
	return ps
}

func (ps *PodSelector) LoadPods(kubeconfigPath, namespace string, app *Application) {
	if kubeconfigPath == "" || namespace == "" {
		ps.pods = []string{}
		ps.filteredPods = []string{}
		ps.clickables = []widget.Clickable{}
		return
	}

	ps.loading = true
	app.isLoading = true
	app.loadingStartTime = time.Now()

	// Asynchronous pod loading
	go func() {
		pods := ps.getPodsFromKubectl(kubeconfigPath, namespace)

		// Проверяем на ошибки сети (если не удалось получить pods)
		if len(pods) == 0 {
			// Проверяем, что это действительно ошибка сети, а не пустой namespace
			// Попробуем простую команду kubectl version для проверки доступности кластера
			homeDir, _ := os.UserHomeDir()
			configPath := filepath.Join(homeDir, ".kube", kubeconfigPath)
			cmd := exec.Command("kubectl", "--kubeconfig", configPath, "version", "--short")
			setSysProcAttr(cmd)
			if err := cmd.Run(); err != nil {
				// Если kubectl version не работает, значит проблема с сетью/кластером
				app.hasNetworkError = true
				app.lastFailedAction = "loadPods"
				ps.loading = false
				app.isLoading = false
				if app.invalidate != nil {
					app.invalidate()
				}
				return
			}
		}

		// Обновляем в UI потоке
		ps.pods = pods
		ps.filteredPods = make([]string, len(pods))
		copy(ps.filteredPods, pods)
		ps.clickables = make([]widget.Clickable, len(pods))

		// Проверяем, что сохраненный pod все еще существует
		saved := ps.selectedPod
		ps.selectedPod = ""
		for _, podName := range pods {
			if podName == saved {
				ps.selectedPod = saved
				break
			}
		}

		ps.loading = false
		app.isLoading = false
		if app.invalidate != nil {
			app.invalidate()
		}
	}()
}

func (ps *PodSelector) getPodsFromKubectl(kubeconfigPath, namespace string) []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}

	configPath := filepath.Join(homeDir, ".kube", kubeconfigPath)

	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "--kubeconfig", configPath, "-o", "jsonpath={.items[*].metadata.name}")
	
	// Устанавливаем атрибуты процесса для скрытия окна терминала (Windows)
	setSysProcAttr(cmd)

	// Захватываем stderr для диагностики
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			log.Printf("Error getting pods: %v | Details: %s", err, stderr.String())
		} else {
			log.Printf("Error getting pods: %v", err)
		}
		return []string{}
	}

	podNames := strings.Fields(string(output))
	sort.Strings(podNames)
	return podNames
}

func (ps *PodSelector) filterPods() {
	if ps.searchText == "" {
		ps.filteredPods = make([]string, len(ps.pods))
		copy(ps.filteredPods, ps.pods)
	} else {
		ps.filteredPods = []string{}
		searchLower := strings.ToLower(ps.searchText)
		for _, pod := range ps.pods {
			if strings.Contains(strings.ToLower(pod), searchLower) {
				ps.filteredPods = append(ps.filteredPods, pod)
			}
		}
		
		// Если после фильтрации ничего не найдено, добавляем "(нет)"
		if len(ps.filteredPods) == 0 {
			ps.filteredPods = []string{"(нет)"}
		}
	}
	ps.clickables = make([]widget.Clickable, len(ps.filteredPods))
}

func (ps *PodSelector) saveSelection() {
	configFile := getPodConfigFilePath()
	os.MkdirAll(filepath.Dir(configFile), 0755)
	os.WriteFile(configFile, []byte(ps.selectedPod), 0644)
}

func (ps *PodSelector) loadSelection() {
	configFile := getPodConfigFilePath()
	data, err := os.ReadFile(configFile)
	if err == nil {
		ps.selectedPod = string(data)
	}
}

func (ps *PodSelector) Layout(gtx layout.Context, th *material.Theme, app *Application) layout.Dimensions {
	// Обновляем фильтрацию при изменении текста поиска
	if ps.searchEditor.Text() != ps.searchText {
		ps.searchText = ps.searchEditor.Text()
		ps.filterPods()
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Фиксированная ширина для метки - 120dp
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
					return material.Label(th, unit.Sp(16), "Pod:").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					// Обработка клика по кнопке
					for ps.button.Clicked(gtx) {
						// Если этот селект уже открыт - просто закрываем его
						if ps.expanded {
							ps.expanded = false
						} else {
							// Если закрыт - закрываем все остальные и открываем этот
							app.closeAllSelectors()
							ps.expanded = true
							// Очищаем поиск при открытии
							ps.searchEditor.SetText("")
							ps.searchText = ""
							ps.filterPods()
						}
					}

					// Pointer cursor on hover
					if ps.button.Hovered() {
						pointer.CursorPointer.Add(gtx.Ops)
					}

					// Button text
					buttonText := "Select pod"
					if ps.selectedPod != "" {
						buttonText = ps.selectedPod
					} else if len(ps.pods) == 0 && !ps.loading {
						buttonText = "First select namespace"
					}

					btn := material.Button(th, &ps.button, buttonText)
					if ps.selectedPod == "" && len(ps.pods) > 0 {
						// Gray color for inactive button
						btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						btn.Color = color.NRGBA{R: 80, G: 80, B: 80, A: 255}
					} else {
						// Normal colors for active button
						btn.Background = color.NRGBA{R: 240, G: 240, B: 240, A: 255}
						btn.Color = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
					}
					return btn.Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !ps.expanded {
				return layout.Dimensions{}
			}

			// Draw border around container
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					// Gray border
					defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
					paint.Fill(gtx.Ops, color.NRGBA{R: 180, G: 180, B: 180, A: 255})
					return layout.Dimensions{Size: gtx.Constraints.Max}
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(1), Right: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// White background inside
						defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				// Поле поиска
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// Фон для поля поиска на всю ширину
						defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, color.NRGBA{R: 248, G: 248, B: 248, A: 255})

						return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							editor := material.Editor(th, &ps.searchEditor, "Search pod...")
							editor.Editor.SingleLine = true
							editor.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
							editor.HintColor = color.NRGBA{R: 120, G: 120, B: 120, A: 255}

							return editor.Layout(gtx)
						})
					})
				}),
				// Список подов - занимает все доступное место
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					if len(ps.filteredPods) == 0 {
						// Если нет результатов поиска, показываем пустое пространство
						return layout.Dimensions{}
					}
					return material.List(th, &ps.list).Layout(gtx, len(ps.filteredPods), func(gtx layout.Context, index int) layout.Dimensions {
						if index >= len(ps.clickables) {
							return layout.Dimensions{}
						}

						// Обработка клика по элементу списка
						for ps.clickables[index].Clicked(gtx) {
							newPod := ps.filteredPods[index]
							
							// Если кликнули по "(нет)" - очищаем выбор
							if newPod == "(нет)" {
								ps.selectedPod = ""
								ps.expanded = false
								ps.saveSelection()
								
								// Очищаем статус записи и кнопку браузера при очистке pod
								if app != nil {
									app.recordingResult = ""
									app.showBrowserButton = false
									app.htmlOutputPath = ""
									app.hasCompletedRecording = false
								}
								break
							}
							
							// Если выбираем тот же pod - просто закрываем селект
							if newPod == ps.selectedPod {
								ps.expanded = false
								break
							}
							
							// Меняем pod
							ps.selectedPod = newPod
							ps.expanded = false
							ps.saveSelection()
							
							// Очищаем статус записи и кнопку браузера при смене pod
							if app != nil {
								app.recordingResult = ""
								app.showBrowserButton = false
								app.htmlOutputPath = ""
								app.hasCompletedRecording = false
							}
						}

						// Стиль элемента
						isSelected := ps.filteredPods[index] == ps.selectedPod

						return material.Clickable(gtx, &ps.clickables[index], func(gtx layout.Context) layout.Dimensions {
							// Фон для выбранного элемента
							if isSelected {
								defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
								paint.Fill(gtx.Ops, color.NRGBA{R: 220, G: 220, B: 220, A: 255})
							}

							// Pointer cursor при наведении
							if ps.clickables[index].Hovered() {
								pointer.CursorPointer.Add(gtx.Ops)
							}

							// Используем Flexed(1) чтобы элемент занял всю ширину
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										label := material.Label(th, unit.Sp(14), ps.filteredPods[index])
										if isSelected {
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
				}),
			)
					})
				},
			)
		}),
	)
}

func (ps *PodSelector) GetSelectedPod() string {
	return ps.selectedPod
}

func (ps *PodSelector) IsPodSelected() bool {
	return ps.selectedPod != ""
}

func (ps *PodSelector) Close() {
	ps.expanded = false
}

func (ps *PodSelector) Reset() {
	ps.pods = []string{}
	ps.filteredPods = []string{}
	ps.selectedPod = ""
	ps.expanded = false
	ps.searchEditor.SetText("")
	ps.searchText = ""
}

func (ps *PodSelector) IsExpanded() bool {
	return ps.expanded
}

func getPodConfigFilePath() string {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".k8s-pfr")
	return filepath.Join(configDir, "pod.mem")
}

func getAsprofArgsConfigFilePath() string {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".k8s-pfr")
	return filepath.Join(configDir, "asprof_args.mem")
}

func getFolderConfigFilePath() string {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".k8s-pfr")
	return filepath.Join(configDir, "jfr_folder.mem")
}

func getFormatConfigFilePath() string {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".k8s-pfr")
	return filepath.Join(configDir, "convert_format.mem")
}

func getConfigDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".k8s-pfr")
}

func shortenFileName(filePath string) string {
	fileName := filepath.Base(filePath)
	if len(fileName) <= 13 { // 5 + 3 + 5 = 13, если меньше - показываем полностью
		return fileName
	}
	
	ext := filepath.Ext(fileName)
	nameWithoutExt := strings.TrimSuffix(fileName, ext)
	
	if len(nameWithoutExt) <= 10 { // 5 + 5 = 10
		return fileName
	}
	
	return nameWithoutExt[:5] + "***" + nameWithoutExt[len(nameWithoutExt)-5:] + ext
}

func clearAllSavedData() error {
	configDir := getConfigDir()
	
	// Удаляем всю директорию конфигурации
	if err := os.RemoveAll(configDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config directory: %v", err)
	}
	
	// Создаем директорию заново
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}
	
	return nil
}

type Application struct {
	versionBadge       widget.Clickable
	version            string
	kubeconfigSelector *KubeconfigSelector
	namespaceSelector  *NamespaceSelector
	podSelector        *PodSelector
	formatSelector     *FormatSelector
	lastSelectedConfig string
	isLoading          bool
	loadingStartTime   time.Time
	asprofArgsEditor   widget.Editor
	asprofArgs         string
	invalidate         func() // function for forced UI refresh
	folderButton       widget.Clickable
	selectedFolder     string
	startRecordingButton widget.Clickable
	isRecording        bool
	recordingResult    string
	profilerPath       string // Путь к файлу профайлера
	openBrowserButton  widget.Clickable
	showBrowserButton  bool // Показывать ли кнопку открытия в браузере
	htmlOutputPath     string // Путь к HTML файлу для открытия
	hasCompletedRecording bool // Была ли завершена запись для текущей конфигурации
	isInitializing     bool   // Идет ли первоначальная инициализация
	initializationMessage string // Сообщение инициализации
	hasError           bool   // Есть ли критическая ошибка
	isClearing         bool   // Идет ли очистка сохраненных данных
	
	// Состояние выбора папки
	isChoosingFolder   bool   // Открыто ли окно выбора папки
	errorMessage       string // Сообщение об ошибке
	statusClickable    widget.Clickable // Кликабельность статуса
	outputPath         string // Путь к папке с результатами
	
	// Состояние сетевых ошибок
	hasNetworkError    bool   // Есть ли ошибка сети
	retryButton        widget.Clickable // Кнопка Retry
	resetButton        widget.Clickable // Кнопка Reset
	lastFailedAction   string // Последнее неудачное действие для повтора
	
	// Логотип приложения
	logoImage          paint.ImageOp // Логотип
	
	// UI компоненты
	headerComponent    *HeaderComponent // Компонент заголовка
}

func (a *Application) closeAllSelectors() {
	a.kubeconfigSelector.expanded = false
	a.namespaceSelector.expanded = false
	a.podSelector.expanded = false
	a.formatSelector.expanded = false
}

func (a *Application) loadLogo() {
	logoPath := filepath.Join("media", "logo_50.png")
	
	file, err := os.Open(logoPath)
	if err != nil {
		log.Printf("Не удалось открыть файл логотипа: %v", err)
		return
	}
	defer file.Close()
	
	img, err := png.Decode(file)
	if err != nil {
		log.Printf("Не удалось декодировать PNG: %v", err)
		return
	}
	
	a.logoImage = paint.NewImageOp(img)
}

func (a *Application) drawLoadingOverlay(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if !a.isLoading {
		return layout.Dimensions{}
	}

	// Создаем кликабельную область на весь экран для блокировки
	clickable := &widget.Clickable{}
	return material.Clickable(gtx, clickable, func(gtx layout.Context) layout.Dimensions {
		// Очень темный полупрозрачный фон (opacity 0.8)
		defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 250}) // 255 * 0.8 = 204

		// Center "Loading" text
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							label := material.Label(th, unit.Sp(24), "Loading...")
							label.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
							return label.Layout(gtx)
						})
					}),
				)
			}),
		)
	})
}

// Функция для отрисовки блокирующего overlay во время записи
func (a *Application) drawRecordingOverlay(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if !a.isRecording {
		return layout.Dimensions{}
	}

	// Создаем кликабельную область на весь экран для блокировки
	clickable := &widget.Clickable{}
	return material.Clickable(gtx, clickable, func(gtx layout.Context) layout.Dimensions {
		// Полупрозрачный фон для блокировки взаимодействия (opacity 0.2)
		defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 51}) // 255 * 0.2 = 51
		
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

func (a *Application) drawFolderChoosingOverlay(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if !a.isChoosingFolder {
		return layout.Dimensions{}
	}

	// Создаем кликабельную область на весь экран для блокировки
	clickable := &widget.Clickable{}
	return material.Clickable(gtx, clickable, func(gtx layout.Context) layout.Dimensions {
		defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 253})

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							label := material.Label(th, unit.Sp(24), "Choosing JFR folder...")
							label.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
							return label.Layout(gtx)
						})
					}),
				)
			}),
		)
	})
}

func (a *Application) drawNetworkErrorOverlay(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if !a.hasNetworkError {
		return layout.Dimensions{}
	}

	// Создаем кликабельную область на весь экран для блокировки
	clickable := &widget.Clickable{}
	return material.Clickable(gtx, clickable, func(gtx layout.Context) layout.Dimensions {
		// Полностью белый фон
		defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
		paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
								// Сообщение об ошибке
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									label := material.Label(th, unit.Sp(20), "Network error. Check connection or enable VPN.")
									label.Color = color.NRGBA{R: 200, G: 50, B: 50, A: 255} // Красный цвет
									label.Alignment = text.Middle
									return label.Layout(gtx)
								}),
								// Отступ между надписью и кнопками
								layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),
								// Кнопки Retry и Reset
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
										// Кнопка Retry
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											// Обработка клика на кнопку Retry
											for a.retryButton.Clicked(gtx) {
												a.retryLastAction()
											}

											// Pointer cursor при наведении
											if a.retryButton.Hovered() {
												pointer.CursorPointer.Add(gtx.Ops)
											}

											btn := material.Button(th, &a.retryButton, "Retry")
											btn.Background = color.NRGBA{R: 76, G: 175, B: 80, A: 255} // Зеленая кнопка
											btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
											return btn.Layout(gtx)
										}),
										// Отступ между кнопками
										layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
										// Кнопка Reset
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											// Обработка клика на кнопку Reset
											for a.resetButton.Clicked(gtx) {
												a.resetAll()
											}

											// Pointer cursor при наведении
											if a.resetButton.Hovered() {
												pointer.CursorPointer.Add(gtx.Ops)
											}

											btn := material.Button(th, &a.resetButton, "Reset")
											btn.Background = color.NRGBA{R: 244, G: 67, B: 54, A: 255} // Красная кнопка
											btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
											return btn.Layout(gtx)
										}),
									)
								}),
							)
						})
					}),
				)
			}),
		)
	})
}

func (a *Application) retryLastAction() {
	a.hasNetworkError = false
	
	switch a.lastFailedAction {
	case "loadNamespaces":
		selectedConfig := a.kubeconfigSelector.GetSelectedConfig()
		if selectedConfig != "" {
			a.namespaceSelector.LoadNamespaces(selectedConfig, a)
		}
	case "loadPods":
		selectedConfig := a.kubeconfigSelector.GetSelectedConfig()
		selectedNamespace := a.namespaceSelector.GetSelectedNamespace()
		if selectedConfig != "" && selectedNamespace != "" {
			a.podSelector.LoadPods(selectedConfig, selectedNamespace, a)
		}
	}
	
	if a.invalidate != nil {
		a.invalidate()
	}
}

func (a *Application) resetAll() {
	// Скрываем сетевую ошибку
	a.hasNetworkError = false
	
	// Очищаем все сохраненные конфигурации
	if err := clearAllSavedData(); err != nil {
		log.Printf("Warning: Failed to clear saved data: %v", err)
	}
	
	// Сбрасываем все селекторы в начальное состояние
	if a.kubeconfigSelector != nil {
		a.kubeconfigSelector.selectedConfig = ""
		a.kubeconfigSelector.expanded = false
		a.kubeconfigSelector.scanKubeconfigs() // Перезагружаем список конфигов
	}
	
	if a.namespaceSelector != nil {
		a.namespaceSelector.Reset()
	}
	
	if a.podSelector != nil {
		a.podSelector.Reset()
	}
	
	// Очищаем статус записи и результаты
	a.recordingResult = ""
	a.showBrowserButton = false
	a.htmlOutputPath = ""
	a.hasCompletedRecording = false
	a.outputPath = ""
	
	// Очищаем поля ввода
	a.asprofArgs = "-e cpu -d 30" // Возвращаем дефолтные значения
	a.asprofArgsEditor.SetText(a.asprofArgs)
	
	// Сбрасываем папку на дефолтную
	homeDir, _ := os.UserHomeDir()
	a.selectedFolder = filepath.Join(homeDir, "Desktop")
	
	if a.invalidate != nil {
		a.invalidate()
	}
}

// Функция для проверки и загрузки необходимых файлов при первом запуске
func checkAndDownloadDependencies() error {
	// Проверяем существование папки ./data
	if _, err := os.Stat("./data"); os.IsNotExist(err) {
		log.Println("Creating ./data directory...")
		if err := os.MkdirAll("./data", 0755); err != nil {
			return fmt.Errorf("failed to create ./data directory: %v", err)
		}
	}

	// Проверяем async-profiler
	profilerPath := "./data/async-profiler-4.1-linux-x64.tar.gz"
	if _, err := os.Stat(profilerPath); os.IsNotExist(err) {
		log.Println("Downloading async-profiler...")
		profilerURL := "https://github.com/async-profiler/async-profiler/releases/download/v4.1/async-profiler-4.1-linux-x64.tar.gz"
		if err := downloadFile(profilerURL, profilerPath); err != nil {
			return fmt.Errorf("failed to download async-profiler: %v", err)
		}
	}

	// Проверяем jfr-converter
	converterPath := "./data/jfr-converter.jar"
	if _, err := os.Stat(converterPath); os.IsNotExist(err) {
		log.Println("Downloading jfr-converter...")
		converterURL := "https://github.com/async-profiler/async-profiler/releases/download/v4.1/jfr-converter.jar"
		if err := downloadFile(converterURL, converterPath); err != nil {
			return fmt.Errorf("failed to download jfr-converter: %v", err)
		}
	}

	// Проверяем команду kubectl
	if err := checkKubectl(); err != nil {
		return fmt.Errorf("kubectl check failed: %v", err)
	}

	// Проверяем папку ~/.kube
	if err := checkKubeDirectory(); err != nil {
		return fmt.Errorf("kube directory check failed: %v", err)
	}

	return nil
}

// Функция для загрузки файла
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Функция для проверки kubectl
func checkKubectl() error {
	cmd := exec.Command("kubectl", "version", "--client=true")
	setSysProcAttr(cmd)
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl not found or not working: %v", err)
	}
	return nil
}

// Функция для проверки папки ~/.kube
func checkKubeDirectory() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	kubeDir := filepath.Join(homeDir, ".kube")
	if _, err := os.Stat(kubeDir); os.IsNotExist(err) {
		return fmt.Errorf("~/.kube directory not found - please set up kubectl first")
	}

	return nil
}

func NewApplication() *Application {
	app := &Application{
		asprofArgsEditor: widget.Editor{
			SingleLine: true,
		},
	}
	app.detectVersion()
	app.loadLogo() // Загружаем логотип
	
	// Создаем компонент заголовка
	app.headerComponent = NewHeaderComponent(app.logoImage, &app.versionBadge, app.version)

	// Проверяем существование data директории
	if _, err := os.Stat("./data"); os.IsNotExist(err) {
		// Если data нет, сначала очищаем данные, затем запускаем инициализацию
		app.isClearing = true
		app.initializationMessage = "Clearing old data..."
		
		// Запускаем очистку в фоне
		go app.performClearing()
	} else {
		// Если data есть, инициализируем селекторы и загружаем как обычно
		app.initializeSelectors()
		app.loadAsprofArgs()     // Load saved arguments
		app.loadSelectedFolder() // Load saved folder
		app.profilerPath = app.findProfilerPath() // Find profiler automatically

		selectedConfig := app.kubeconfigSelector.GetSelectedConfig()
		if selectedConfig != "" {
			go func() {
				app.namespaceSelector.LoadNamespaces(selectedConfig, app)

				// Загружаем поды если есть сохраненный namespace
				selectedNamespace := app.namespaceSelector.GetSelectedNamespace()
				if selectedNamespace != "" {
					app.podSelector.LoadPods(selectedConfig, selectedNamespace, app)
				}
			}()
		}
	}

	return app
}

func (a *Application) initializeSelectors() {
	a.kubeconfigSelector = NewKubeconfigSelector()
	a.namespaceSelector = NewNamespaceSelector()
	a.podSelector = NewPodSelector()
	a.formatSelector = NewFormatSelector()
}

func (a *Application) performClearing() {
	// Очищаем все сохраненные данные
	if err := clearAllSavedData(); err != nil {
		log.Printf("Warning: Failed to clear saved data: %v", err)
	}
	
	// После очистки переходим к инициализации
	a.isClearing = false
	a.isInitializing = true
	a.initializationMessage = "Loading async-profiler '4.1'..."
	
	if a.invalidate != nil {
		a.invalidate()
	}
	
	// Запускаем обычную инициализацию
	a.performInitialization()
}

func (a *Application) performInitialization() {
	// Проверяем kubectl и .kube перед загрузкой зависимостей
	if err := checkKubectl(); err != nil {
		a.hasError = true
		a.errorMessage = "kubectl not found"
		a.isInitializing = false
		if a.invalidate != nil {
			a.invalidate()
		}
		return
	}
	
	if err := checkKubeDirectory(); err != nil {
		a.hasError = true
		a.errorMessage = ".kube directory not found or empty"
		a.isInitializing = false
		if a.invalidate != nil {
			a.invalidate()
		}
		return
	}

	// Загружаем зависимости
	if err := checkAndDownloadDependencies(); err != nil {
		log.Printf("Warning: Failed to check dependencies: %v", err)
	}
	
	// После загрузки инициализируем селекторы и загружаем данные
	a.initializeSelectors()
	a.loadAsprofArgs()
	a.loadSelectedFolder()
	a.profilerPath = a.findProfilerPath()

	selectedConfig := a.kubeconfigSelector.GetSelectedConfig()
	if selectedConfig != "" {
		a.namespaceSelector.LoadNamespaces(selectedConfig, a)

		selectedNamespace := a.namespaceSelector.GetSelectedNamespace()
		if selectedNamespace != "" {
			a.podSelector.LoadPods(selectedConfig, selectedNamespace, a)
		}
	}
	
	// Завершаем инициализацию
	a.isInitializing = false
	if a.invalidate != nil {
		a.invalidate()
	}
}

func (a *Application) loadAsprofArgs() {
	configFile := getAsprofArgsConfigFilePath()
	data, err := os.ReadFile(configFile)
	if err == nil {
		savedArgs := strings.TrimSpace(string(data))
		a.asprofArgs = savedArgs
		a.asprofArgsEditor.SetText(savedArgs)
	} else {
		// Устанавливаем дефолтные значения если файл не найден
		defaultArgs := "-e cpu -d 30"
		a.asprofArgs = defaultArgs
		a.asprofArgsEditor.SetText(defaultArgs)
		// Сохраняем дефолтные значения
		os.MkdirAll(filepath.Dir(configFile), 0755)
		os.WriteFile(configFile, []byte(defaultArgs), 0644)
	}
}

func (a *Application) saveAsprofArgs() {
	if a.asprofArgs == "" {
		return
	}
	configFile := getAsprofArgsConfigFilePath()
	os.MkdirAll(filepath.Dir(configFile), 0755)
	os.WriteFile(configFile, []byte(a.asprofArgs), 0644)
}

func (a *Application) loadSelectedFolder() {
	configFile := getFolderConfigFilePath()
	data, err := os.ReadFile(configFile)
	if err == nil {
		a.selectedFolder = strings.TrimSpace(string(data))
	} else {
		// Папка по умолчанию
		homeDir, _ := os.UserHomeDir()
		a.selectedFolder = filepath.Join(homeDir, "Desktop")
	}
}

func (a *Application) saveSelectedFolder() {
	configFile := getFolderConfigFilePath()
	os.MkdirAll(filepath.Dir(configFile), 0755)
	os.WriteFile(configFile, []byte(a.selectedFolder), 0644)
}

// Функция для автоматического поиска файла профайлера
func (a *Application) findProfilerPath() string {
	// Ищем в папке data рядом с исполняемым файлом
	dataDir := "./data"
	if entries, err := os.ReadDir(dataDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), "async-profiler") && strings.HasSuffix(entry.Name(), ".tar.gz") {
				return filepath.Join(dataDir, entry.Name())
			}
		}
	}
	
	// Если не найден, возвращаем стандартный путь
	return "./data/async-profiler-4.1-linux-x64.tar.gz"
}

func (a *Application) drawFolderSelector(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Строка с меткой и кнопкой
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Фиксированная ширина для метки - 120dp
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
					return material.Label(th, unit.Sp(16), "JFR Folder:").Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					// Обработка клика по кнопке (только если не записываем и не выбираем папку)
					for a.folderButton.Clicked(gtx) && !a.isRecording && !a.isChoosingFolder {
						// Показываем занавес и запускаем нативный диалог
						a.isChoosingFolder = true
						if a.invalidate != nil {
							a.invalidate()
						}
						
						go func() {
							if folder, err := chooseFolderDialog(); err == nil {
								a.selectedFolder = folder
								a.saveSelectedFolder()
							}
							// Убираем занавес в любом случае
							a.isChoosingFolder = false
							if a.invalidate != nil {
								a.invalidate()
							}
						}()
					}

					// Pointer cursor при наведении (только если не записываем)
					if !a.isRecording && a.folderButton.Hovered() {
						pointer.CursorPointer.Add(gtx.Ops)
					}

					btn := material.Button(th, &a.folderButton, "Select Folder")
					if a.isRecording {
						// Серая кнопка во время записи
						btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						btn.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
					} else {
						// Обычные цвета
						btn.Background = color.NRGBA{R: 240, G: 240, B: 240, A: 255}
						btn.Color = color.NRGBA{R: 50, G: 50, B: 50, A: 255}
					}
					return btn.Layout(gtx)
				}),
			)
		}),
		// Отображение выбранной папки
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.selectedFolder == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(130)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Label(th, unit.Sp(12), a.selectedFolder)
				label.Color = color.NRGBA{R: 80, G: 80, B: 80, A: 255}
				return label.Layout(gtx)
			})
		}),
	)
}

// Функция для отрисовки кнопки записи и результата
func (a *Application) drawRecordingControls(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		// Пустое пространство слева
		layout.Flexed(1, layout.Spacer{}.Layout),
		// Правая панель с кнопками и статусом (вертикально)
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.End}.Layout(gtx,
				// Кнопка "Start Recording"
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Устанавливаем фиксированную ширину для кнопки
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(150))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(150))

					// Обработка клика по кнопке
					for a.startRecordingButton.Clicked(gtx) {
						if !a.isRecording {
							a.startRecording()
						}
					}

					// Pointer cursor при наведении (только если не записываем)
					if !a.isRecording && a.startRecordingButton.Hovered() {
						pointer.CursorPointer.Add(gtx.Ops)
					}

					buttonText := "Start Recording"
					if a.isRecording {
						buttonText = "Recording..."
					} else if a.hasCompletedRecording {
						buttonText = "Restart Recording"
					}

					btn := material.Button(th, &a.startRecordingButton, buttonText)
					if a.isRecording {
						// Серая кнопка во время записи
						btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						btn.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
					} else {
						// Зеленая кнопка для записи
						btn.Background = color.NRGBA{R: 76, G: 175, B: 80, A: 255}
						btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
					}
					return btn.Layout(gtx)
				}),
				// Отступ между кнопкой и статусом
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				// Результат записи
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if a.recordingResult == "" {
						return layout.Dimensions{}
					}
					
					// Проверяем клик по статусу "Saved"
					if strings.Contains(a.recordingResult, "Saved") && a.outputPath != "" {
						for a.statusClickable.Clicked(gtx) {
							go a.openFolder(a.outputPath)
						}
						
						// Pointer cursor при наведении на статус "Saved"
						if a.statusClickable.Hovered() {
							pointer.CursorPointer.Add(gtx.Ops)
						}
						
						return material.Clickable(gtx, &a.statusClickable, func(gtx layout.Context) layout.Dimensions {
							label := material.Label(th, unit.Sp(12), a.recordingResult)
							if strings.Contains(a.recordingResult, "Error") {
								label.Color = color.NRGBA{R: 200, G: 50, B: 50, A: 255} // Красный для ошибок
							} else {
								label.Color = color.NRGBA{R: 50, G: 150, B: 50, A: 255} // Зеленый для успеха (подчеркиваем что кликабельно)
							}
							return label.Layout(gtx)
						})
					} else {
						// Обычный статус (не кликабельный)
						label := material.Label(th, unit.Sp(12), a.recordingResult)
						if strings.Contains(a.recordingResult, "Error") {
							label.Color = color.NRGBA{R: 200, G: 50, B: 50, A: 255} // Красный для ошибок
						} else {
							label.Color = color.NRGBA{R: 50, G: 150, B: 50, A: 255} // Зеленый для успеха
						}
						return label.Layout(gtx)
					}
				}),
				// Отступ между статусом и кнопкой браузера
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				// Кнопка "Open in Browser" для HTML/heatmap файлов
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !a.showBrowserButton || a.htmlOutputPath == "" {
						return layout.Dimensions{}
					}
					// Устанавливаем фиксированную ширину для кнопки (такую же как у кнопки записи)
					gtx.Constraints.Min.X = gtx.Dp(unit.Dp(150))
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(150))
					
					// Обработка клика по кнопке браузера
					for a.openBrowserButton.Clicked(gtx) {
						go a.openInBrowser(a.htmlOutputPath)
					}

					// Pointer cursor при наведении
					if a.openBrowserButton.Hovered() {
						pointer.CursorPointer.Add(gtx.Ops)
					}

					btn := material.Button(th, &a.openBrowserButton, "Open in Browser")
					btn.Background = color.NRGBA{R: 33, G: 150, B: 243, A: 255} // Синяя кнопка
					btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
					return btn.Layout(gtx)
				}),
			)
		}),
	)
}

func (a *Application) drawAsprofArgsInput(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Обновляем значение при изменении текста и сохраняем
	if a.asprofArgsEditor.Text() != a.asprofArgs {
		a.asprofArgs = a.asprofArgsEditor.Text()
		a.saveAsprofArgs() // Сохраняем при каждом изменении
	}

	// Ограничиваем высоту поля ввода (как у кнопок селекторов)
	maxHeight := gtx.Dp(unit.Dp(48)) // примерно высота кнопки селектора
	if gtx.Constraints.Max.Y > maxHeight {
		gtx.Constraints.Max.Y = maxHeight
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// Фиксированная ширина для метки - 120dp (как у других селекторов)
			gtx.Constraints.Min.X = gtx.Dp(unit.Dp(120))
			gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
			return material.Label(th, unit.Sp(16), "Arguments:").Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			// Используем тот же стиль что и у кнопок селекторов
			editor := material.Editor(th, &a.asprofArgsEditor, "Enter arguments for asprof")
			editor.Editor.SingleLine = true
			editor.Color = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
			editor.HintColor = color.NRGBA{R: 120, G: 120, B: 120, A: 255}

			// Добавляем фон и отступы как у кнопок
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					// Тот же фон что у кнопок селекторов
					defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()
					paint.Fill(gtx.Ops, color.NRGBA{R: 240, G: 240, B: 240, A: 255})
					return layout.Dimensions{Size: gtx.Constraints.Max}
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return editor.Layout(gtx)
					})
				},
			)
		}),
	)
}

func (a *Application) detectVersion() {
	// Ищем файлы async-profiler*.tar.gz прямо в data/
	files, err := filepath.Glob(filepath.Join("data", "async-profiler*.tar.gz"))
	if err != nil || len(files) == 0 {
		// Если файлы не найдены, показываем версию по умолчанию
		a.version = "4.1"
		return
	}

	// Берем первый найденный файл и извлекаем версию
	filename := filepath.Base(files[0])
	// async-profiler-4.1-linux-x64.tar.gz -> 4.1
	parts := strings.Split(filename, "-")
	if len(parts) >= 3 {
		a.version = parts[2] // версия находится на 3-й позиции
	} else {
		a.version = "unknown"
	}
}

func (a *Application) onVersionBadgeClicked() {
	url := "https://github.com/async-profiler/async-profiler"
	log.Printf("Открываем GitHub: %s", url)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		setSysProcAttr(cmd)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		log.Printf("Неподдерживаемая ОС: %s", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Ошибка открытия браузера: %v", err)
	}
}

func (a *Application) openInBrowser(filePath string) {
	log.Printf("Открываем файл в браузере: %s", filePath)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", "file:///"+strings.ReplaceAll(filePath, "\\", "/"))
		setSysProcAttr(cmd)
	case "darwin":
		cmd = exec.Command("open", filePath)
	case "linux":
		cmd = exec.Command("xdg-open", filePath)
	default:
		log.Printf("Неподдерживаемая ОС: %s", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Ошибка открытия файла: %v", err)
	}
}

func (a *Application) openFolder(folderPath string) {
	log.Printf("Открываем папку: %s", folderPath)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", folderPath)
	case "darwin":
		cmd = exec.Command("open", folderPath)
	case "linux":
		cmd = exec.Command("xdg-open", folderPath)
	default:
		log.Printf("Неподдерживаемая ОС: %s", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Ошибка открытия папки: %v", err)
	}
}

// Функция для выполнения kubectl команд с правильным KUBECONFIG
func runKubectlWithConfig(kubeconfigPath string, args ...string) error {
	cmd := exec.Command("kubectl", args...)
	// Временно захватываем stderr для диагностики
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	
	// Устанавливаем атрибуты процесса для скрытия окна терминала (Windows)
	setSysProcAttr(cmd)
	
	err := cmd.Run()
	if err != nil && stderr.Len() > 0 {
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return err
}

// Функция для копирования файла
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

// Функция для запуска профилирования в отдельной горутине
func (a *Application) startRecording() {
	go func() {
		// Устанавливаем состояние записи
		a.isRecording = true
		a.recordingResult = ""
		a.showBrowserButton = false // Скрываем кнопку браузера при новой записи
		a.htmlOutputPath = ""       // Очищаем путь к HTML файлу
		a.invalidate()

		// Получаем параметры
		selectedConfig := a.kubeconfigSelector.GetSelectedConfig()
		selectedNamespace := a.namespaceSelector.GetSelectedNamespace()
		selectedPod := a.podSelector.GetSelectedPod()
		asprofArgs := a.asprofArgs
		outputFolder := a.selectedFolder

		// Создаем временный kubeconfig
		home, err := os.UserHomeDir()
		if err != nil {
			a.recordingResult = fmt.Sprintf("Error getting home directory: %v", err)
			a.isRecording = false
			a.invalidate()
			return
		}

		kubeconfigPath := filepath.Join(home, ".kube", selectedConfig)
		data, err := os.ReadFile(kubeconfigPath)
		if err != nil {
			a.recordingResult = fmt.Sprintf("Error reading kubeconfig: %v", err)
			a.isRecording = false
			a.invalidate()
			return
		}

		// Создаем временный файл kubeconfig
		homeDir, _ := os.UserHomeDir()
		tmpDir := filepath.Join(homeDir, ".k8s-pfr", "tmp")
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			a.recordingResult = fmt.Sprintf("Error creating temp directory: %v", err)
			a.isRecording = false
			a.invalidate()
			return
		}
		defer os.RemoveAll(tmpDir)

		tempKubeconfigPath := filepath.Join(tmpDir, "kubeconfig.yaml")
		if err := os.WriteFile(tempKubeconfigPath, data, 0644); err != nil {
			a.recordingResult = fmt.Sprintf("Error writing temp kubeconfig: %v", err)
			a.isRecording = false
			a.invalidate()
			return
		}

		// Устанавливаем KUBECONFIG для этого процесса
		originalKubeconfig := os.Getenv("KUBECONFIG")
		os.Setenv("KUBECONFIG", tempKubeconfigPath)
		defer os.Setenv("KUBECONFIG", originalKubeconfig)

		// Пути для профайлера
		profilerTar := a.profilerPath
		remoteDir := "/tmp/async-profiler-4.1-linux-x64"
		remoteTar := "/tmp/async-profiler-4.1-linux-x64.tar.gz"
		remoteJfr := "/tmp/recording.jfr"

		// Аргументы для namespace
		nsArgs := []string{}
		if selectedNamespace != "" {
			nsArgs = append(nsArgs, "-n", selectedNamespace)
		}

		// Проверяем наличие профайлера в поде
		checkArgs := append(nsArgs, "exec", selectedPod, "--", "bash", "-c", fmt.Sprintf("[ -d %s ]", remoteDir))
		checkCmd := exec.Command("kubectl", checkArgs...)
		checkCmd.Env = append(os.Environ(), "KUBECONFIG="+tempKubeconfigPath)
		
		// Захватываем stderr для подробной информации
		var checkStderr bytes.Buffer
		checkCmd.Stderr = &checkStderr
		
		// Устанавливаем атрибуты процесса для скрытия окна терминала (Windows)
		setSysProcAttr(checkCmd)

		if err := checkCmd.Run(); err != nil {
			// Показываем статус копирования
			a.recordingResult = "Copying profiler..."
			a.invalidate()
			
			// Копируем профайлер в под
			copyArgs := append(nsArgs, "cp", profilerTar, fmt.Sprintf("%s:%s", selectedPod, remoteTar))
			if err := runKubectlWithConfig(tempKubeconfigPath, copyArgs...); err != nil {
				a.recordingResult = fmt.Sprintf("Error copying profiler: %v", err)
				a.isRecording = false
				a.invalidate()
				return
			}

			// Показываем статус извлечения
			a.recordingResult = "Extracting profiler..."
			a.invalidate()

			// Извлекаем профайлер
			extractArgs := append(nsArgs, "exec", selectedPod, "--", "bash", "-c", fmt.Sprintf("tar xzf %s -C /tmp", remoteTar))
			if err := runKubectlWithConfig(tempKubeconfigPath, extractArgs...); err != nil {
				a.recordingResult = fmt.Sprintf("Error extracting profiler: %v", err)
				a.isRecording = false
				a.invalidate()
				return
			}
		}

		// Показываем статус запуска профайлера
		a.recordingResult = "Starting profiler..."
		a.invalidate()

		// Запускаем профайлер
		profilerCmd := fmt.Sprintf("%s/bin/asprof -f %s %s 1", remoteDir, remoteJfr, asprofArgs)
		execArgs := append(nsArgs, "exec", selectedPod, "--", "bash", "-c", profilerCmd)
		if err := runKubectlWithConfig(tempKubeconfigPath, execArgs...); err != nil {
			a.recordingResult = fmt.Sprintf("Error running profiler: %v", err)
			a.isRecording = false
			a.invalidate()
			return
		}

		// Показываем статус копирования результата
		a.recordingResult = "Copying result..."
		a.invalidate()

		// Создаем имя файла с timestamp
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("%s__%s__%s.jfr", selectedNamespace, selectedPod, timestamp)
		
		// Создаем целевую папку если она не существует
		outputPath := filepath.Join(outputFolder, filename)
		if err := os.MkdirAll(outputFolder, 0755); err != nil {
			a.recordingResult = fmt.Sprintf("Error creating output folder: %v", err)
			a.isRecording = false
			a.invalidate()
			return
		}

		// Сначала копируем из пода в текущую папку (рядом с исполняемым файлом)
		localTempFile := "./" + filename
		
		var sourceSpec string
		if selectedNamespace != "" {
			sourceSpec = fmt.Sprintf("%s/%s:%s", selectedNamespace, selectedPod, remoteJfr)
		} else {
			sourceSpec = fmt.Sprintf("%s:%s", selectedPod, remoteJfr)
		}
		
		copyResultArgs := []string{"cp", sourceSpec, localTempFile}
		if err := runKubectlWithConfig(tempKubeconfigPath, copyResultArgs...); err != nil {
			a.recordingResult = fmt.Sprintf("Error copying result: %v", err)
			a.isRecording = false
			a.invalidate()
			return
		}

		// Перемещаем из текущей папки в целевую папку
		if err := os.Rename(localTempFile, outputPath); err != nil {
			a.recordingResult = fmt.Sprintf("Error moving file: %v", err)
			a.isRecording = false
			a.invalidate()
			return
		}

		// Конвертируем JFR если выбран формат
		selectedFormat := a.formatSelector.GetSelectedFormat()
		if selectedFormat != "" && selectedFormat != "(none)" {
			a.recordingResult = "Converting JFR..."
			a.invalidate()

			// Копируем JFR файл обратно в текущую папку для конвертации
			if err := copyFile(outputPath, localTempFile); err != nil {
				a.recordingResult = fmt.Sprintf("Error preparing for conversion: %v", err)
				a.isRecording = false
				a.invalidate()
				return
			}

			// Запускаем конвертер
			converterPath := "./data/jfr-converter.jar"
			convertCmd := exec.Command("java", "-jar", converterPath, "-o", selectedFormat, localTempFile)
			
			// Устанавливаем атрибуты процесса для скрытия окна терминала (Windows)
			setSysProcAttr(convertCmd)
			
			// Захватываем stderr для подробной информации об ошибках
			var stderr bytes.Buffer
			convertCmd.Stderr = &stderr
			
			if err := convertCmd.Run(); err != nil {
				errMsg := fmt.Sprintf("Error converting JFR: %v", err)
				if stderr.Len() > 0 {
					errMsg += fmt.Sprintf(" | Details: %s", stderr.String())
				}
				a.recordingResult = errMsg
				a.isRecording = false
				os.Remove(localTempFile) // Удаляем временный файл
				a.invalidate()
				return
			}

			// jfr-converter создает файл с тем же именем как входной, но с другим расширением
			// Ищем любой файл с базовым именем JFR файла
			baseNameWithoutExt := strings.TrimSuffix(localTempFile, ".jfr")
			pattern := baseNameWithoutExt + ".*"
			
			files, err := filepath.Glob(pattern)
			if err != nil || len(files) == 0 {
				a.recordingResult = "Error: converted file not found"
				a.isRecording = false
				os.Remove(localTempFile)
				a.invalidate()
				return
			}
			
			// Находим файл который НЕ JFR (созданный конвертером)
			var convertedFile string
			for _, file := range files {
				if !strings.HasSuffix(file, ".jfr") {
					convertedFile = file
					break
				}
			}
			
			if convertedFile == "" {
				a.recordingResult = "Error: no converted file found (only JFR)"
				a.isRecording = false
				os.Remove(localTempFile)
				a.invalidate()
				return
			}
			
			// Получаем расширение созданного файла
			actualExt := filepath.Ext(convertedFile)
			
			// Создаем правильное имя для выходного файла
			baseFilename := fmt.Sprintf("%s__%s__%s", selectedNamespace, selectedPod, timestamp)
			finalOutputPath := filepath.Join(outputFolder, baseFilename+actualExt)
			if err := os.Rename(convertedFile, finalOutputPath); err != nil {
				a.recordingResult = fmt.Sprintf("Error moving converted file: %v", err)
				a.isRecording = false
				os.Remove(localTempFile) // Удаляем временный файл
				a.invalidate()
				return
			}

			// Сохраняем путь к HTML файлу для кнопки браузера и определяем showBrowserButton
			if actualExt == ".html" || selectedFormat == "heatmap" || selectedFormat == "html" {
				a.htmlOutputPath = finalOutputPath
			}

			// Удаляем временный JFR файл
			os.Remove(localTempFile)

			a.outputPath = outputFolder // Сохраняем путь для кликабельности
		} else {
			a.outputPath = filepath.Dir(outputPath) // Сохраняем папку с файлом
		}

		// Очищаем временные файлы в поде
		a.recordingResult = "Cleaning up..."
		a.invalidate()
		
		// Удаляем файлы и папку профилировщика
		cleanupArgs := append(nsArgs, "exec", selectedPod, "--", "rm", "-rf", remoteJfr, remoteTar, remoteDir)
		runKubectlWithConfig(tempKubeconfigPath, cleanupArgs...) // Игнорируем ошибки очистки

		// Завершение
		if selectedFormat != "" && selectedFormat != "(none)" {
			a.recordingResult = fmt.Sprintf("Saved JFR and %s files to %s", selectedFormat, outputFolder)
			a.outputPath = outputFolder // Сохраняем путь для кликабельности
		} else {
			a.recordingResult = fmt.Sprintf("Saved JFR to %s", outputFolder)
			a.outputPath = filepath.Dir(outputPath) // Сохраняем папку с файлом
		}
		a.isRecording = false
		a.hasCompletedRecording = true // Помечаем что запись завершена

		if selectedFormat == "heatmap" || selectedFormat == "html" {
			a.showBrowserButton = true
		}
		
		a.invalidate()
	}()
}

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("k8s-pfr.beta"))
		w.Option(app.NavigationColor(color.NRGBA {R: 46, G: 108, B: 230, A: 255}))
		w.Option(app.StatusColor(color.NRGBA {R: 46, G: 108, B: 230, A: 255}))
		w.Option(app.Size(unit.Dp(800), unit.Dp(790)))
		w.Option(app.MinSize(unit.Dp(800), unit.Dp(790)))
		w.Option(app.MaxSize(unit.Dp(800), unit.Dp(790)))

		err := run(w)
		if err != nil {
			log.Fatal(err)
		}
	}()
	app.Main()
}

func run(w *app.Window) error {
	th := material.NewTheme()
	var ops op.Ops

	appInstance := NewApplication()
	appInstance.invalidate = w.Invalidate

	// Функция закрытия окна - больше не нужна, окно закрывается стандартными способами
	// closeWindow := func() {
	//     w.Perform(system.ActionClose)
	// }

	w.Invalidate()

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			// Закрываем любые открытые диалоги
			closeAnyOpenDialogs()
			// Принудительное завершение программы при закрытии окна
			os.Exit(0)
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// Основной layout
			layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					// Если есть критическая ошибка - показываем окно ошибки
					if appInstance.hasError {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							// Заголовок вверху
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return appInstance.headerComponent.SimpleHeaderLayout(gtx, th, nil)
							}),
							// Центрированное сообщение об ошибке
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
											layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												label := material.Label(th, unit.Sp(18), appInstance.errorMessage)
												label.Color = color.NRGBA{R: 255, G: 0, B: 0, A: 255} // Красный цвет для ошибки
												label.Alignment = text.Middle
												return label.Layout(gtx)
											}),
										)
									}),
								)
							}),
						)
					}

					// Если идет очистка или инициализация - показываем только заголовок и лоадер
					if appInstance.isClearing || appInstance.isInitializing {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							// Заголовок вверху
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return appInstance.headerComponent.SimpleHeaderLayout(gtx, th, nil)
							}),
							// Центрированное сообщение о загрузке
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
											layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												label := material.Label(th, unit.Sp(18), appInstance.initializationMessage)
												label.Color = th.Palette.Fg
												label.Alignment = text.Middle
												return label.Layout(gtx)
											}),
										)
									}),
								)
							}),
						)
					}

					// Обычный UI
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						// Верхняя панель с названием и версией
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							// Обрабатываем клики на версии
							for appInstance.versionBadge.Clicked(gtx) {
								appInstance.onVersionBadgeClicked()
							}
							// Устанавливаем pointer cursor при наведении
							if appInstance.versionBadge.Hovered() {
								pointer.CursorPointer.Add(gtx.Ops)
							}
							
							return appInstance.headerComponent.Layout(gtx, th, nil)
						}),
						// Основное содержимое - используем Flexed для растягивания
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: unit.Dp(20), Bottom: unit.Dp(20), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								// Проверяем изменение kubeconfig для загрузки namespaces
								var selectedConfig string
								if appInstance.kubeconfigSelector != nil {
									selectedConfig = appInstance.kubeconfigSelector.GetSelectedConfig()
								}

								// Если kubeconfig изменился, сбрасываем namespaces
								if selectedConfig != appInstance.lastSelectedConfig {
									appInstance.lastSelectedConfig = selectedConfig
									if selectedConfig == "" && appInstance.namespaceSelector != nil {
										appInstance.namespaceSelector.Reset()
									}
								}

								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									// Kubeconfig selector
									func() layout.FlexChild {
										if appInstance.kubeconfigSelector == nil {
											return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return layout.Dimensions{}
											})
										}
										if appInstance.kubeconfigSelector.expanded {
											// Если kubeconfig расширен - он занимает максимум места
											return layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												return appInstance.kubeconfigSelector.Layout(gtx, th, appInstance)
											})
										} else {
											// Обычный размер
											return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return appInstance.kubeconfigSelector.Layout(gtx, th, appInstance)
											})
										}
									}(),
									layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
									// Namespace selector
									func() layout.FlexChild {
										if selectedConfig == "" || appInstance.namespaceSelector == nil {
											return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return layout.Dimensions{}
											})
										}
										if appInstance.namespaceSelector.expanded {
											// Если namespace расширен - он занимает максимум места
											return layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												return appInstance.namespaceSelector.Layout(gtx, th, appInstance)
											})
										} else {
											// Обычный размер
											return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return appInstance.namespaceSelector.Layout(gtx, th, appInstance)
											})
										}
									}(),
									layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
									// Pod selector
									func() layout.FlexChild {
										var selectedNamespace string
										if appInstance.namespaceSelector != nil {
											selectedNamespace = appInstance.namespaceSelector.GetSelectedNamespace()
										}
										if selectedConfig == "" || selectedNamespace == "" || appInstance.podSelector == nil {
											return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return layout.Dimensions{}
											})
										}
										if appInstance.podSelector.expanded {
											// Если pod расширен - он занимает максимум места
											return layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
												return appInstance.podSelector.Layout(gtx, th, appInstance)
											})
										} else {
											// Обычный размер
											return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												return appInstance.podSelector.Layout(gtx, th, appInstance)
											})
										}
									}(),
									layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
									// Поле ввода аргументов async-profiler
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										var selectedNamespace, selectedPod string
										if appInstance.namespaceSelector != nil {
											selectedNamespace = appInstance.namespaceSelector.GetSelectedNamespace()
										}
										if appInstance.podSelector != nil {
											selectedPod = appInstance.podSelector.GetSelectedPod()
										}
										if selectedConfig == "" || selectedNamespace == "" || selectedPod == "" {
											return layout.Dimensions{}
										}
										return appInstance.drawAsprofArgsInput(gtx, th)
									}),
									layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
									// Селектор папки для JFR
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										selectedNamespace := appInstance.namespaceSelector.GetSelectedNamespace()
										selectedPod := appInstance.podSelector.GetSelectedPod()
										if selectedConfig == "" || selectedNamespace == "" || selectedPod == "" {
											return layout.Dimensions{}
										}
										return appInstance.drawFolderSelector(gtx, th)
									}),
									layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
									// Селектор формата JFR
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										var selectedNamespace, selectedPod string
										if appInstance.namespaceSelector != nil {
											selectedNamespace = appInstance.namespaceSelector.GetSelectedNamespace()
										}
										if appInstance.podSelector != nil {
											selectedPod = appInstance.podSelector.GetSelectedPod()
										}
										if selectedConfig == "" || selectedNamespace == "" || selectedPod == "" || appInstance.formatSelector == nil {
											return layout.Dimensions{}
										}
										return appInstance.formatSelector.Layout(gtx, th, appInstance)
									}),
								)
							})
						}),
						// Кнопка записи внизу экрана (зафиксирована)
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							var selectedConfig, selectedNamespace, selectedPod string
							if appInstance.kubeconfigSelector != nil {
								selectedConfig = appInstance.kubeconfigSelector.GetSelectedConfig()
							}
							if appInstance.namespaceSelector != nil {
								selectedNamespace = appInstance.namespaceSelector.GetSelectedNamespace()
							}
							if appInstance.podSelector != nil {
								selectedPod = appInstance.podSelector.GetSelectedPod()
							}
							
							if selectedConfig == "" || selectedNamespace == "" || selectedPod == "" || appInstance.selectedFolder == "" {
								return layout.Dimensions{}
							}
							
							return layout.Inset{Top: unit.Dp(20), Bottom: unit.Dp(20), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return appInstance.drawRecordingControls(gtx, th)
							})
						}),
					)
				}),
				// Overlay с лоадером
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return appInstance.drawLoadingOverlay(gtx, th)
				}),
				// Блокирующий overlay во время записи
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return appInstance.drawRecordingOverlay(gtx, th)
				}),
				// Занавес во время выбора папки
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return appInstance.drawFolderChoosingOverlay(gtx, th)
				}),
				// Overlay для ошибок сети
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return appInstance.drawNetworkErrorOverlay(gtx, th)
				}),
			)

			e.Frame(gtx.Ops)
		}
	}
}
