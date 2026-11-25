//go:build !cli

/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	_ "embed"
	"image/color"
	"math"
	"runtime"

	"github.com/AllenDang/cimgui-go/imgui"
	g "github.com/AllenDang/giu"
)

//go:embed fonts/Inter-Regular.ttf
var interRegularBytes []byte

//go:embed fonts/Inter-SemiBold.ttf
var interSemiBoldBytes []byte

// Vencord.dev dark theme palette (Gruvbox Material Dark)
// Directly matched to https://vencord.dev CSS custom properties.
var (
	// Backgrounds
	ThemeBgDim         = color.RGBA{0x14, 0x16, 0x17, 0xff} // --dark-bgDim: #141617
	ThemeBg0           = color.RGBA{0x1d, 0x20, 0x21, 0xff} // --dark-bg0: #1d2021
	ThemeBg1           = color.RGBA{0x28, 0x28, 0x28, 0xff} // --dark-bg1: #282828
	ThemeBgCurrentWord = color.RGBA{0x32, 0x30, 0x2f, 0xff} // --dark-bgCurrentWord: #32302f
	ThemeBg3           = color.RGBA{0x3c, 0x38, 0x36, 0xff} // --dark-bg3 / --dark-bg4: #3c3836
	ThemeBg5           = color.RGBA{0x50, 0x49, 0x44, 0xff} // --dark-bg5: #504944

	// Subtle tinted surface backgrounds for callouts & states
	ThemeBgVisualYellow = color.RGBA{0x47, 0x3c, 0x29, 0xff} // --dark-bgVisualYellow: #473c29
	ThemeBgVisualGreen  = color.RGBA{0x33, 0x3e, 0x34, 0xff} // --dark-bgVisualGreen: #333e34
	ThemeBgVisualBlue   = color.RGBA{0x2e, 0x3b, 0x3b, 0xff} // --dark-bgVisualBlue: #2e3b3b
	ThemeBgVisualRed    = color.RGBA{0x44, 0x2e, 0x2d, 0xff} // --dark-bgVisualRed: #442e2d

	// Foregrounds & neutrals
	ThemeFg0       = color.RGBA{0xd4, 0xbe, 0x98, 0xff} // --dark-fg0: #d4be98
	ThemeFg1       = color.RGBA{0xdd, 0xc7, 0xa0, 0xff} // --dark-fg1: #ddc7a0
	ThemeFg0Muted  = color.RGBA{0xd4, 0xbe, 0x98, 0x66} // --dark-fg0-muted: #d4be9866
	ThemeFg0Muted2 = color.RGBA{0xd4, 0xbe, 0x98, 0x25} // --dark-fg0-muted2: #d4be9825
	ThemeGrey0     = color.RGBA{0x7c, 0x6f, 0x64, 0xff} // --dark-grey0: #7c6f64
	ThemeGrey1     = color.RGBA{0x91, 0x82, 0x74, 0xff} // --dark-grey1: #918274
	ThemeGrey2     = color.RGBA{0xa7, 0x9a, 0x83, 0xff} // --dark-grey2: #a79a83

	// Accents
	ThemeAccentPurple = color.RGBA{0xd3, 0x86, 0x9b, 0xff} // --dark-accentPurple: #d3869b
	ThemeAccentRed    = color.RGBA{0xea, 0x69, 0x62, 0xff} // --dark-accentRed: #ea6962
	ThemeAccentOrange = color.RGBA{0xe7, 0x8a, 0x4f, 0xff} // --dark-accentOrange: #e78a4f
	ThemeAccentYellow = color.RGBA{0xd8, 0xa6, 0x56, 0xff} // --dark-accentYellow: #d8a656
	ThemeAccentGreen  = color.RGBA{0xa8, 0xb6, 0x65, 0xff} // --dark-accentGreen: #a8b665
	ThemeAccentAqua   = color.RGBA{0x89, 0xb4, 0x81, 0xff} // --dark-accentAqua: #89b481
	ThemeAccentBlue   = color.RGBA{0x7c, 0xae, 0xa3, 0xff} // --dark-accentBlue: #7caea3

	// Derived overlay & state colors
	ThemeModalDimBg = color.RGBA{0x14, 0x16, 0x17, 0xc0}
	ThemeSelection  = color.RGBA{0xd3, 0x86, 0x9b, 0x4d}
)

const (
	minZoomScale     float32 = 0.70
	maxZoomScale     float32 = 2.00
	defaultZoomScale float32 = 1.00
	zoomScaleStep    float32 = 0.10
)

var (
	zoomScale float32 = defaultZoomScale

	interRegular  *g.FontInfo
	interSemiBold *g.FontInfo

	cardHeights = map[string]float32{}
)

// snap rounds a coordinate to the nearest integer pixel to avoid bilinear font blur.
func snap(v float32) float32 {
	return float32(math.Round(float64(v)))
}

// scaled scales a UI dimension by the active zoom factor and snaps to whole pixels.
func scaled(v float32) float32 {
	return snap(v * zoomScale)
}

func initFonts() {
	atlas := g.Context.FontAtlas
	atlas.SetDefaultFontFromBytes(interRegularBytes)
	atlas.SetDefaultFontSize(20)
	interRegular = atlas.AddFontFromBytes("Inter-Regular", interRegularBytes)
	interSemiBold = atlas.AddFontFromBytes("Inter-SemiBold", interSemiBoldBytes)
}

type textFont struct {
	face *g.FontInfo
	size float32
}

func (f textFont) style() *g.StyleSetter {
	return g.Style().SetFont(f.face).SetFontSize(scaled(f.size))
}

func fontRegular(size float32) textFont {
	return textFont{interRegular, size}
}

func fontSemiBold(size float32) textFont {
	return textFont{interSemiBold, size}
}

// setZoomScale updates zoomScale clamped to [minZoomScale, maxZoomScale] and rounded to 2 decimal places.
func setZoomScale(next float32) bool {
	rounded := float32(math.Round(float64(next)*100) / 100)
	if rounded < minZoomScale {
		rounded = minZoomScale
	} else if rounded > maxZoomScale {
		rounded = maxZoomScale
	}
	if rounded == zoomScale {
		return false
	}
	zoomScale = rounded
	return true
}

func applyZoom() {
	cardHeights = map[string]float32{}
	g.Update()
}

// blendColor linearly blends foreground c over background bg using factor t in [0, 1].
func blendColor(c, bg color.RGBA, t float32) color.RGBA {
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	inv := 1 - t
	return color.RGBA{
		R: uint8(float32(c.R)*t + float32(bg.R)*inv),
		G: uint8(float32(c.G)*t + float32(bg.G)*inv),
		B: uint8(float32(c.B)*t + float32(bg.B)*inv),
		A: 0xff,
	}
}

// withAlpha returns a copy of c with the given alpha byte.
func withAlpha(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}

// accentHover matches vencord.dev's 80% hover blend on dark backgrounds.
func accentHover(c color.RGBA) color.RGBA {
	return blendColor(c, ThemeBg1, 0.82)
}

// accentActive matches vencord.dev's pressed state blend.
func accentActive(c color.RGBA) color.RGBA {
	return blendColor(c, ThemeBg0, 0.70)
}

// applyTheme applies the vencord.dev dark theme globally to ImGui and the master window.
func applyTheme(w *g.MasterWindow) {
	// giu now applies monitor DPI to all platforms. On macOS and Linux the
	// installer uses window coordinates and ImGui handles framebuffer density.
	if runtime.GOOS != "windows" {
		style := imgui.NewStyle()
		imgui.CurrentContext().SetStyle(*style)
		style.Destroy()
	}

	w.SetBgColor(ThemeBg1)

	style := g.Style()
	set := style.SetColor

	// Text
	set(g.StyleColorText, ThemeFg0)
	set(g.StyleColorTextDisabled, ThemeGrey0)
	set(g.StyleColorTextSelectedBg, ThemeSelection)

	// Window & surface backgrounds
	set(g.StyleColorWindowBg, ThemeBg1)
	set(g.StyleColorChildBg, ThemeBgCurrentWord)
	set(g.StyleColorPopupBg, ThemeBg0)
	set(g.StyleColorMenuBarBg, ThemeBg0)

	// Borders & separators
	set(g.StyleColorBorder, ThemeFg0Muted2)
	set(g.StyleColorBorderShadow, color.RGBA{0, 0, 0, 0})
	set(g.StyleColorSeparator, ThemeFg0Muted2)
	set(g.StyleColorSeparatorHovered, ThemeAccentPurple)
	set(g.StyleColorSeparatorActive, ThemeAccentPurple)

	// Frames (inputs, checkboxes, radio circles)
	set(g.StyleColorFrameBg, ThemeBg3)
	set(g.StyleColorFrameBgHovered, ThemeBg5)
	set(g.StyleColorFrameBgActive, ThemeBgCurrentWord)

	// Title bars
	set(g.StyleColorTitleBg, ThemeBg0)
	set(g.StyleColorTitleBgActive, ThemeBg0)
	set(g.StyleColorTitleBgCollapsed, ThemeBgDim)

	// Scrollbars
	set(g.StyleColorScrollbarBg, withAlpha(ThemeBg0, 0x80))
	set(g.StyleColorScrollbarGrab, ThemeBg3)
	set(g.StyleColorScrollbarGrabHovered, ThemeBg5)
	set(g.StyleColorScrollbarGrabActive, ThemeGrey0)

	// Interactive controls (checkmarks, sliders, default secondary buttons)
	set(g.StyleColorCheckMark, ThemeAccentPurple)
	set(g.StyleColorSliderGrab, ThemeAccentPurple)
	set(g.StyleColorSliderGrabActive, ThemeFg1)

	set(g.StyleColorButton, ThemeBg3)
	set(g.StyleColorButtonHovered, ThemeBg5)
	set(g.StyleColorButtonActive, ThemeBgCurrentWord)

	// Headers & selectables
	set(g.StyleColorHeader, ThemeBg3)
	set(g.StyleColorHeaderHovered, ThemeBg5)
	set(g.StyleColorHeaderActive, withAlpha(ThemeAccentPurple, 0x45))

	// Resize grips
	set(g.StyleColorResizeGrip, withAlpha(ThemeAccentPurple, 0x40))
	set(g.StyleColorResizeGripHovered, withAlpha(ThemeAccentPurple, 0x99))
	set(g.StyleColorResizeGripActive, ThemeAccentPurple)

	// Tabs
	set(g.StyleColorTab, ThemeBg0)
	set(g.StyleColorTabHovered, ThemeBg5)
	set(g.StyleColorTabActive, ThemeBg3)
	set(g.StyleColorTabUnfocused, ThemeBg0)
	set(g.StyleColorTabUnfocusedActive, ThemeBgCurrentWord)

	// Plots & tables
	set(g.StyleColorPlotLines, ThemeAccentBlue)
	set(g.StyleColorPlotLinesHovered, ThemeAccentPurple)
	set(g.StyleColorPlotHistogram, ThemeAccentYellow)
	set(g.StyleColorPlotHistogramHovered, ThemeAccentOrange)
	set(g.StyleColorTableHeaderBg, ThemeBg0)
	set(g.StyleColorTableBorderStrong, ThemeFg0Muted)
	set(g.StyleColorTableBorderLight, ThemeFg0Muted2)

	// Navigation & modal dimming
	set(g.StyleColorDragDropTarget, ThemeAccentYellow)
	set(g.StyleColorID(imgui.ColNavCursor), ThemeAccentPurple)
	set(g.StyleColorNavWindowingHighlight, ThemeFg1)
	set(g.StyleColorNavWindowingDimBg, ThemeModalDimBg)
	set(g.StyleColorModalWindowDimBg, ThemeModalDimBg)
	w.SetStyle(style)
}

// pushGlobalStyleVars applies consistent vencord.dev rounding, border, and spacing vars for the frame.
// Returns the number of pushed style vars to pass to g.PopStyleV.
func pushGlobalStyleVars() int {
	imgui.PushStyleVarFloat(imgui.StyleVarWindowRounding, 0)
	imgui.PushStyleVarFloat(imgui.StyleVarChildRounding, scaled(14))
	imgui.PushStyleVarFloat(imgui.StyleVarPopupRounding, scaled(12))
	imgui.PushStyleVarFloat(imgui.StyleVarFrameRounding, scaled(10))
	imgui.PushStyleVarFloat(imgui.StyleVarGrabRounding, scaled(8))
	imgui.PushStyleVarFloat(imgui.StyleVarScrollbarRounding, scaled(8))
	imgui.PushStyleVarFloat(imgui.StyleVarScrollbarSize, scaled(10))
	imgui.PushStyleVarFloat(imgui.StyleVarFrameBorderSize, 0)
	imgui.PushStyleVarVec2(imgui.StyleVarItemSpacing, imgui.Vec2{X: scaled(10), Y: scaled(8)})
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{X: scaled(36), Y: scaled(28)})
	return 10
}

// accentButton renders a filled accent button with dark high-contrast text, matching vencord.dev's primary button style.
func accentButton(label string, accent color.RGBA, width, height float32, disabled bool, tip string, onClick func()) g.Widget {
	widgets := g.Layout{
		g.Button(label).
			OnClick(onClick).
			Size(width, height),
	}
	if tip != "" {
		widgets = append(widgets, Tooltip(tip))
	}

	return g.Style().
		SetColor(g.StyleColorButton, accent).
		SetColor(g.StyleColorButtonHovered, accentHover(accent)).
		SetColor(g.StyleColorButtonActive, accentActive(accent)).
		SetColor(g.StyleColorText, ThemeBg0).
		SetStyleFloat(g.StyleVarFrameRounding, scaled(10)).
		SetStyleFloat(g.StyleVarFrameBorderSize, 0).
		SetDisabled(disabled).
		To(widgets...)
}

// secondaryButton renders an outlined dark button matching vencord.dev's secondary action buttons.
func secondaryButton(label string, width, height float32, disabled bool, tip string, onClick func()) g.Widget {
	widgets := g.Layout{
		g.Button(label).
			OnClick(onClick).
			Size(width, height),
	}
	if tip != "" {
		widgets = append(widgets, Tooltip(tip))
	}

	return g.Style().
		SetColor(g.StyleColorButton, ThemeBg3).
		SetColor(g.StyleColorButtonHovered, ThemeBg5).
		SetColor(g.StyleColorButtonActive, ThemeBgCurrentWord).
		SetColor(g.StyleColorBorder, ThemeFg0Muted).
		SetColor(g.StyleColorText, ThemeFg0).
		SetStyleFloat(g.StyleVarFrameRounding, scaled(10)).
		SetStyleFloat(g.StyleVarFrameBorderSize, 1.5).
		SetDisabled(disabled).
		To(widgets...)
}

// buildInlineActionLink renders a compact clickable link styled in vencord.dev's accentBlue
// and returns the rendered button width.
func buildInlineActionLink(label, tip string, padX, padY float32, onClick func()) float32 {
	var width float32
	g.Style().
		SetColor(g.StyleColorButton, color.RGBA{0, 0, 0, 0}).
		SetColor(g.StyleColorButtonHovered, withAlpha(ThemeAccentBlue, 0x28)).
		SetColor(g.StyleColorButtonActive, withAlpha(ThemeAccentBlue, 0x40)).
		SetColor(g.StyleColorText, ThemeAccentBlue).
		SetStyleFloat(g.StyleVarFrameRounding, scaled(6)).
		SetStyleFloat(g.StyleVarFrameBorderSize, 0).
		SetStyle(g.StyleVarFramePadding, padX, padY).
		To(g.Custom(func() {
			g.Button(label).OnClick(onClick).Build()
			width = imgui.ItemRectSize().X
			if g.IsItemHovered() {
				g.SetMouseCursor(g.MouseCursorHand)
				if tip != "" {
					Tooltip(tip).Build()
				}
			}
		})).Build()
	return width
}

// alignedRowItem renders one item at explicit integer coordinates and returns its rendered width.
type alignedRowItem struct {
	render func(x, rowY, rowH float32) float32
}

// renderAlignedRow places items horizontally in a single row and centers every item vertically
// against rowH without calling SameLine, avoiding ImGui's CurrLineTextBaseOffset baseline drift.
func renderAlignedRow(rowH float32, itemGap float32, items ...alignedRowItem) g.Widget {
	return g.Custom(func() {
		if len(items) == 0 {
			return
		}
		startPos := imgui.CursorPos()
		startX := snap(startPos.X)
		rowY := snap(startPos.Y)
		rowH = snap(rowH)

		curX := startX
		for i, item := range items {
			if i > 0 {
				curX += itemGap
			}
			itemW := item.render(snap(curX), rowY, rowH)
			curX += itemW
		}

		// Move cursor cleanly to the bottom of the row; Dummy(0, 0) lets ImGui apply ItemSpacing.Y once.
		imgui.SetCursorPos(imgui.Vec2{X: startX, Y: rowY + rowH})
		imgui.Dummy(imgui.Vec2{X: 0, Y: 0})
	})
}

func rowAlignedLabel(text string, col color.Color, font textFont) alignedRowItem {
	return alignedRowItem{
		render: func(x, rowY, rowH float32) float32 {
			var itemW float32
			font.style().
				SetColor(g.StyleColorText, col).
				To(g.Custom(func() {
					_, textH := g.CalcTextSize(text)
					y := snap(rowY + (rowH-textH)/2)
					if y < rowY {
						y = rowY
					}
					imgui.SetCursorPos(imgui.Vec2{X: x, Y: y})
					g.Label(text).Build()
					itemW = imgui.ItemRectSize().X
				})).Build()
			return itemW
		},
	}
}

func rowAlignedRadio(label string, active bool, font textFont, onChange func()) alignedRowItem {
	return alignedRowItem{
		render: func(x, rowY, rowH float32) float32 {
			var itemW float32
			font.style().
				To(g.Custom(func() {
					_, framePadY := g.GetFramePadding()
					_, textH := g.CalcTextSize("Ag")
					radioH := textH + framePadY*2
					y := snap(rowY + (rowH-radioH)/2)
					if y < rowY {
						y = rowY
					}
					imgui.SetCursorPos(imgui.Vec2{X: x, Y: y})
					g.RadioButton(label, active).OnChange(onChange).Build()
					itemW = imgui.ItemRectSize().X
				})).Build()
			return itemW
		},
	}
}

func rowAlignedLink(label, tip string, font textFont, padX, padY float32, onClick func()) alignedRowItem {
	return alignedRowItem{
		render: func(x, rowY, rowH float32) float32 {
			var itemW float32
			font.style().
				To(g.Custom(func() {
					_, textH := g.CalcTextSize(label)
					btnH := textH + padY*2
					y := snap(rowY + (rowH-btnH)/2)
					if y < rowY {
						y = rowY
					}
					imgui.SetCursorPos(imgui.Vec2{X: x, Y: y})
					itemW = buildInlineActionLink(label, tip, padX, padY, onClick)
				})).Build()
			return itemW
		},
	}
}

// renderAutoHeightCard renders a styled Child card whose height self-measures from its actual
// rendered contents for exact symmetric top/bottom padding, optionally clamping to maxHeight.
func renderAutoHeightCard(
	id string,
	bgColor, borderColor color.RGBA,
	borderSize, rounding, padX, padY, fallbackHeight, maxHeight float32,
	content ...g.Widget,
) g.Widget {
	return g.Custom(func() {
		measured, ok := cardHeights[id]
		height := fallbackHeight
		if ok && measured > 0 {
			height = measured
		}

		needsScroll := false
		if maxHeight > 0 && height > maxHeight {
			height = maxHeight
			needsScroll = true
		}

		flags := g.WindowFlags(0)
		if !needsScroll {
			flags = g.WindowFlagsNoScrollbar | g.WindowFlagsNoScrollWithMouse
		}

		g.Style().
			SetColor(g.StyleColorChildBg, bgColor).
			SetColor(g.StyleColorBorder, borderColor).
			SetStyle(g.StyleVarWindowPadding, padX, padY).
			SetStyleFloat(g.StyleVarChildRounding, rounding).
			SetStyleFloat(g.StyleVarChildBorderSize, borderSize).
			To(
				g.Child().
					ID(g.ID(id)).
					Border(borderSize > 0).
					Flags(flags).
					Size(g.Auto, height).
					Layout(
						g.Custom(func() {
							startY := imgui.CursorPos().Y
							g.Layout(content).Build()
							endY := imgui.CursorPos().Y
							_, spacingY := g.GetItemSpacing()

							contentH := endY - startY - spacingY
							if contentH < 0 {
								contentH = 0
							}
							nextH := float32(math.Ceil(float64(contentH + padY*2)))
							if math.Abs(float64(nextH-cardHeights[id])) >= 1.0 {
								cardHeights[id] = nextH
								g.Update()
							}
						}),
					),
			).Build()
	})
}

// renderCalloutCard renders a vencord.dev-style callout box with a 2px accent border,
// tinted dark background, and exact symmetric top/bottom padding.
func renderCalloutCard(id string, borderColor, bgColor color.RGBA, title, body string, extra g.Widget) g.Widget {
	content := g.Layout{}
	if title != "" {
		content = append(content,
			fontSemiBold(20).style().
				SetColor(g.StyleColorText, borderColor).
				To(g.Label(title).Wrapped(true)),
		)
	}
	if body != "" {
		content = append(content,
			fontRegular(20).style().
				SetColor(g.StyleColorText, ThemeFg1).
				To(g.Label(body).Wrapped(true)),
		)
	}
	if extra != nil {
		content = append(content, g.Dummy(0, scaled(4)), extra)
	}

	return renderAutoHeightCard(
		id,
		bgColor,
		borderColor,
		2,
		scaled(12),
		scaled(18),
		scaled(14),
		scaled(110),
		0,
		content...,
	)
}
