//go:build !cli

/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	path "path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"vencordinstaller/buildinfo"

	g "github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
	"github.com/go-gl/glfw/v3.3/glfw"

	// png decoder for icon
	_ "image/png"
)

const (
	VencordRepoUrl   = "https://github.com/Vendicated/Vencord"
	InstallerRepoUrl = "https://github.com/Vencord/Installer"
)

var (
	discords        []any
	radioIdx        int
	customChoiceIdx int

	customDir              string
	autoCompleteDir        string
	autoCompleteFile       string
	autoCompleteCandidates []string
	autoCompleteIdx        int
	lastAutoComplete       string
	didAutoComplete        bool

	modalId          = 0
	modalTitle       = "Oh No :("
	modalMessage     = "You should never see this"
	modalExtra       = ""
	modalSupportLink = ""

	acceptedOpenAsar   bool
	showedUpdatePrompt bool
	showCustomLocation bool

	githubDone atomic.Bool
	appIcon    image.Image

	win *g.MasterWindow

	relaunchTarget   *DiscordInstall
	relaunchSnapshot *RunningDiscord
	relaunchNote     string
	relaunchErr      error
)

//go:embed winres/icon.png
var iconBytes []byte

func init() {
	LogLevel = LevelDebug
}

type zoomAction int

const (
	zoomNone zoomAction = iota
	zoomIn
	zoomOut
	zoomReset
)

func hasZoomModifier(mod g.Modifier, goos string) bool {
	// Mask out lock keys so shortcuts still work when NumLock or CapsLock is active.
	clean := mod &^ (g.ModNumLock | g.ModCapsLock)
	// Ignore Alt combinations so AltGr (Ctrl+Alt on Windows) never triggers zoom.
	if clean&g.ModAlt != 0 {
		return false
	}
	if clean&g.ModControl != 0 {
		return true
	}
	if goos == "darwin" && clean&g.ModSuper != 0 {
		return true
	}
	return false
}

func classifyZoomKey(key g.Key, keyName string) zoomAction {
	// Numpad keys are layout-independent across keyboard layouts.
	switch key {
	case g.KeyKPAdd:
		return zoomIn
	case g.KeyKPSubtract:
		return zoomOut
	case g.KeyKP0:
		return zoomReset
	}

	// Layout-aware key name from GLFW (handles QWERTZ, AZERTY, Nordic, US QWERTY, etc.).
	switch keyName {
	case "+", "=":
		return zoomIn
	case "-":
		return zoomOut
	case "0":
		return zoomReset
	}

	// Fallback if the platform did not provide a layout key name.
	if keyName == "" {
		switch key {
		case g.KeyEqual:
			return zoomIn
		case g.KeyMinus:
			return zoomOut
		case g.Key0:
			return zoomReset
		}
	}

	return zoomNone
}

func performZoomAction(action zoomAction) bool {
	switch action {
	case zoomIn:
		if setZoomScale(zoomScale + zoomScaleStep) {
			applyZoom()
			return true
		}
	case zoomOut:
		if setZoomScale(zoomScale - zoomScaleStep) {
			applyZoom()
			return true
		}
	case zoomReset:
		if setZoomScale(defaultZoomScale) {
			applyZoom()
			return true
		}
	}
	return false
}

type zoomInputHandler struct {
	base g.InputHandler
}

func (h *zoomInputHandler) RegisterKeyboardShortcuts(s ...g.Shortcut) {
	if h.base != nil {
		h.base.RegisterKeyboardShortcuts(s...)
	}
}

func (h *zoomInputHandler) UnregisterWindowShortcuts() {
	if h.base != nil {
		h.base.UnregisterWindowShortcuts()
	}
}

func (h *zoomInputHandler) Handle(key g.Key, mod g.Modifier) {
	if hasZoomModifier(mod, runtime.GOOS) {
		keyName := ""
		if key != g.KeyUnknown {
			keyName = glfw.GetKeyName(glfw.Key(key), 0)
		}
		if action := classifyZoomKey(key, keyName); action != zoomNone {
			performZoomAction(action)
			return
		}
	}

	if h.base != nil {
		// Strip lock flags so giu's exact-match shortcut table works with NumLock/CapsLock on.
		cleanMod := mod &^ (g.ModNumLock | g.ModCapsLock)
		h.base.Handle(key, cleanMod)
	}
}

func handleWheelZoom() {
	ctrlHeld := g.IsKeyDown(g.KeyLeftControl) || g.IsKeyDown(g.KeyRightControl)
	if runtime.GOOS == "darwin" {
		ctrlHeld = ctrlHeld || g.IsKeyDown(g.KeyLeftSuper) || g.IsKeyDown(g.KeyRightSuper)
	}
	altHeld := g.IsKeyDown(g.KeyLeftAlt) || g.IsKeyDown(g.KeyRightAlt)
	if !ctrlHeld || altHeld {
		return
	}

	wheel := imgui.CurrentIO().GetMouseWheelDelta()
	if wheel > 0 {
		performZoomAction(zoomIn)
	} else if wheel < 0 {
		performZoomAction(zoomOut)
	}
}

func main() {
	InitGithubDownloader()
	refreshDiscords()

	go func() {
		<-GithubDoneChan
		githubDone.Store(true)
		g.Update()
	}()

	go func() {
		<-SelfUpdateCheckDoneChan
		g.Update()
	}()

	setupDefaultFont()
	win = g.NewMasterWindow("Vencord Installer", 1200, 800, 0)
	win.SetSizeLimits(800, 580, g.DontCare, g.DontCare)
	win.SetInputHandler(&zoomInputHandler{base: g.Context.InputHandler})
	applyTheme(win)
	initFonts()

	icon, _, err := image.Decode(bytes.NewReader(iconBytes))
	if err != nil {
		Log.Warn("Failed to load application icon", err)
		Log.Debug(iconBytes, len(iconBytes))
	} else {
		appIcon = icon
		win.SetIcon([]image.Image{icon})
	}

	win.Run(loop)
}

func retryGithubFetch() {
	if !githubDone.CompareAndSwap(true, false) {
		return
	}

	FetchGithubRelease()
	doneCh := GithubDoneChan
	g.Update()

	go func() {
		<-doneCh
		githubDone.Store(true)
		g.Update()
	}()
}

func refreshDiscords() {
	prev := getSelectedDiscord()
	wasCustom := showCustomLocation && radioIdx == customChoiceIdx

	discords = FindDiscords()
	customChoiceIdx = len(discords)

	if wasCustom {
		radioIdx = customChoiceIdx
	} else if _, idx := findMatchingInstall(discords, prev); idx != -1 {
		radioIdx = idx
	}
	clampRadioIdx()
}

func maxRadioIdx() int {
	if showCustomLocation {
		return customChoiceIdx
	}
	if len(discords) > 0 {
		return len(discords) - 1
	}
	return 0
}

func clampRadioIdx() {
	maxIdx := maxRadioIdx()
	if radioIdx < 0 {
		radioIdx = 0
	} else if radioIdx > maxIdx {
		radioIdx = maxIdx
	}
}

func hasValidSelection() bool {
	if len(discords) > 0 && radioIdx >= 0 && radioIdx < len(discords) {
		return true
	}
	return showCustomLocation && strings.TrimSpace(customDir) != ""
}

func getSelectedDiscord() *DiscordInstall {
	if radioIdx >= 0 && radioIdx < len(discords) {
		return discords[radioIdx].(*DiscordInstall)
	}
	return nil
}

type CondWidget struct {
	predicate  bool
	ifWidget   func() g.Widget
	elseWidget func() g.Widget
}

func (w *CondWidget) Build() {
	if w.predicate {
		if w.ifWidget != nil {
			w.ifWidget().Build()
		}
	} else if w.elseWidget != nil {
		w.elseWidget().Build()
	}
}

func getChosenInstall() *DiscordInstall {
	if prev := getSelectedDiscord(); prev != nil {
		refreshDiscords()
		if refreshed, idx := findMatchingInstall(discords, prev); refreshed != nil {
			radioIdx = idx
			return refreshed
		}
		return nil
	}

	if !showCustomLocation {
		return nil
	}

	if discord := ParseDiscord(customDir, ""); discord != nil {
		return discord
	}

	if discord := ParseDiscordNew(customDir, "", strings.Contains(customDir, "com.discordapp")); discord != nil {
		return discord
	}

	g.OpenPopup("#invalid-custom-location")
	return nil
}

func InstallLatestBuilds() (err error) {
	if IsDevInstall {
		return
	}

	err = installLatestBuilds()
	if err != nil {
		ShowErrorModal("Failed to install the latest Vencord builds from GitHub", err.Error(), "https://vencord.dev/support/#Failed-to-install-the-latest-Vencord-builds-from-GitHub")
	}
	return
}

func handlePatch() {
	choice := getChosenInstall()
	if choice != nil {
		choice.Patch()
	}
}

func handleUnpatch() {
	choice := getChosenInstall()
	if choice != nil {
		choice.Unpatch()
	}
}

func handleOpenAsar() {
	choice := getChosenInstall()
	if choice == nil {
		return
	}
	if acceptedOpenAsar || choice.IsOpenAsar() {
		handleOpenAsarConfirmed(choice)
		return
	}

	g.OpenPopup("#openasar-confirm")
}

func preparePostPatchRelaunch(di *DiscordInstall) {
	relaunchTarget = nil
	relaunchSnapshot = nil
	relaunchNote = ""

	if di == nil {
		return
	}

	name := branchDisplayName(di.branch)
	if runtime.GOOS == "windows" {
		relaunched, err := di.RelaunchKilled()
		if err != nil {
			Log.Warn("Failed to relaunch", name+":", err)
			relaunchNote = "Failed to automatically relaunch " + name + ": " + err.Error() + "\nPlease start it manually."
		} else if relaunched {
			relaunchNote = name + " has been automatically relaunched."
		}
		return
	}

	if running := di.FindRunning(); running != nil {
		relaunchTarget = di
		relaunchSnapshot = running
	}
}

func handleOpenAsarConfirmed(choice *DiscordInstall) {
	if choice == nil {
		return
	}
	if choice.IsOpenAsar() {
		if err := choice.UninstallOpenAsar(); err != nil {
			choice.ClearPendingRelaunch()
			handleErr(choice, err, "uninstall OpenAsar from")
		} else {
			preparePostPatchRelaunch(choice)
			g.OpenPopup("#openasar-unpatched")
			g.Update()
		}
	} else {
		if err := choice.InstallOpenAsar(); err != nil {
			choice.ClearPendingRelaunch()
			handleErr(choice, err, "install OpenAsar on")
		} else {
			preparePostPatchRelaunch(choice)
			g.OpenPopup("#openasar-patched")
			g.Update()
		}
	}
}

func handleErr(di *DiscordInstall, err error, action string) {
	if errors.Is(err, os.ErrPermission) {
		switch runtime.GOOS {
		case "windows":
			err = errors.New("Permission denied. Make sure your Discord is fully closed (from the tray)!")
		case "darwin":
			HandleInsufficientPermissions()
			return
		default:
			err = errors.New("Permission denied. Maybe try running me as Administrator/Root?")
		}
	}

	ShowErrorModal("Failed to "+action+" this Install.", err.Error(), "https://vencord.dev/support/#Failed-to-patch-this-Install")
}

func HandleScuffedInstall() {
	g.OpenPopup("#scuffed-install")
}

func HandleInsufficientPermissions() {
	g.OpenPopup("#insufficient-permissions")
}

func (di *DiscordInstall) Patch() {
	if CheckScuffedInstall() {
		return
	}
	if err := di.patch(); err != nil {
		di.ClearPendingRelaunch()
		handleErr(di, err, "patch")
	} else {
		preparePostPatchRelaunch(di)
		g.OpenPopup("#patched")
	}
}

func (di *DiscordInstall) Unpatch() {
	if err := di.unpatch(); err != nil {
		di.ClearPendingRelaunch()
		handleErr(di, err, "unpatch")
	} else {
		preparePostPatchRelaunch(di)
		g.OpenPopup("#unpatched")
	}
}

func onCustomInputChanged() {
	p := customDir
	if len(p) != 0 {
		// Select the custom option for people
		radioIdx = customChoiceIdx
	}

	if p == "" {
		autoCompleteDir = ""
		autoCompleteFile = ""
		autoCompleteCandidates = nil
		autoCompleteIdx = 0
		lastAutoComplete = ""
		didAutoComplete = false
		return
	}

	dir := path.Dir(p)

	isNewDir := strings.HasSuffix(p, "/") || strings.HasSuffix(p, string(os.PathSeparator))
	wentUpADir := !isNewDir && dir != autoCompleteDir

	if isNewDir || wentUpADir {
		autoCompleteDir = dir
		// reset all the funnies
		autoCompleteIdx = 0
		lastAutoComplete = ""
		autoCompleteFile = ""
		autoCompleteCandidates = nil

		// Generate autocomplete items
		files, err := os.ReadDir(dir)
		if err == nil {
			for _, file := range files {
				autoCompleteCandidates = append(autoCompleteCandidates, file.Name())
			}
		}
	} else if !didAutoComplete {
		// reset auto complete and update our file
		autoCompleteFile = path.Base(p)
		lastAutoComplete = ""
	}

	if wentUpADir {
		autoCompleteFile = path.Base(p)
	}

	didAutoComplete = false
}

// go can you give me []any?
// to pass to giu RangeBuilder?
// yeeeeees
// actually returns []string like a boss
func makeAutoComplete() []any {
	input := strings.ToLower(autoCompleteFile)

	var candidates []any
	for _, e := range autoCompleteCandidates {
		file := strings.ToLower(e)
		if autoCompleteFile == "" || strings.HasPrefix(file, input) {
			candidates = append(candidates, e)
		}
	}
	return candidates
}

func makeRadioOnChange(i int) func() {
	return func() {
		radioIdx = i
	}
}

func vencordCommitUrl(hash string) string {
	return VencordRepoUrl + "/commit/" + hash
}

func installerCommitUrl(hash string) string {
	return InstallerRepoUrl + "/commit/" + hash
}

func renderFilesDirErr() g.Widget {
	return g.Layout{
		g.Dummy(0, scaled(32)),
		renderCalloutCard(
			"FilesDirErrCard",
			ThemeAccentRed,
			ThemeBgVisualRed,
			"Failed to initialize Vencord data directory",
			"Error: Failed to create "+FilesDirErr.Error()+"\nResolve this error, then restart Vencord Installer.",
			nil,
		),
	}
}

func Tooltip(label string) g.Widget {
	return g.Style().
		SetColor(g.StyleColorPopupBg, ThemeBg0).
		SetColor(g.StyleColorBorder, ThemeFg0Muted).
		SetColor(g.StyleColorText, ThemeFg1).
		SetFont(fontRegular(18)).
		SetStyle(g.StyleVarWindowPadding, scaled(10), scaled(8)).
		SetStyleFloat(g.StyleVarWindowRounding, scaled(8)).
		SetStyleFloat(g.StyleVarPopupBorderSize, 1).
		To(
			g.Tooltip(label),
		)
}

func styledModal(id string, content ...g.Widget) g.Widget {
	return g.Style().
		SetColor(g.StyleColorPopupBg, ThemeBgCurrentWord).
		SetColor(g.StyleColorBorder, ThemeFg0Muted).
		SetFont(fontRegular(20)).
		SetStyle(g.StyleVarWindowPadding, scaled(28), scaled(24)).
		SetStyleFloat(g.StyleVarWindowRounding, scaled(14)).
		SetStyleFloat(g.StyleVarPopupBorderSize, 1.5).
		To(
			g.PopupModal(id).
				Flags(g.WindowFlagsNoTitleBar | g.WindowFlagsAlwaysAutoResize).
				Layout(
					g.Custom(func() {
						imgui.PushTextWrapPosV(scaled(620))
						g.Layout(content).Build()
						imgui.PopTextWrapPos()
					}),
				),
		)
}

func InfoModal(id, title, description string) g.Widget {
	return RawInfoModal(id, title, description, "", false)
}

func InfoModalExtra(id, title, description, extra string) g.Widget {
	return RawInfoModal(id, title, description, extra, false)
}

func ErrorModal(id, title, errText, supportLink string) g.Widget {
	return styledModal(
		id,
		g.Style().
			SetColor(g.StyleColorText, ThemeAccentRed).
			SetFont(fontSemiBold(30)).
			To(g.Label(title)),
		g.Dummy(0, scaled(6)),
		g.Custom(func() {
			linkFont := fontSemiBold(20)
			labelFont := fontRegular(20)
			padX := scaled(3)
			padY := scaled(2)
			var rowH float32
			g.Style().SetFont(labelFont).To(g.Custom(func() {
				_, textH := g.CalcTextSize("Ag")
				rowH = textH + padY*2
			})).Build()
			renderAlignedRow(
				rowH,
				scaled(2),
				rowAlignedLabel("If this issue persists, visit", ThemeFg0, labelFont),
				rowAlignedLink("our support page", supportLink, linkFont, padX, padY, func() {
					g.OpenURL(supportLink)
				}),
			).Build()
		}),
		g.Dummy(0, scaled(14)),
		g.Style().
			SetColor(g.StyleColorText, ThemeGrey2).
			SetFont(fontSemiBold(18)).
			To(g.Label("Error Information")),
		g.Dummy(0, scaled(4)),
		g.Style().
			SetColor(g.StyleColorChildBg, ThemeBg0).
			SetColor(g.StyleColorBorder, ThemeFg0Muted2).
			SetStyle(g.StyleVarWindowPadding, scaled(14), scaled(12)).
			SetStyleFloat(g.StyleVarChildRounding, scaled(10)).
			SetStyleFloat(g.StyleVarChildBorderSize, 1).
			To(
				g.Child().
					Border(true).
					Size(scaled(580), scaled(130)).
					Layout(
						g.Style().
							SetColor(g.StyleColorText, ThemeFg1).
							SetFont(fontRegular(18)).
							To(g.Label(errText).Wrapped(true)),
					),
			),
		g.Dummy(0, scaled(16)),
		g.Style().SetFont(fontSemiBold(20)).To(
			accentButton("Ok", ThemeAccentPurple, scaled(120), scaled(40), false, "", func() {
				g.CloseCurrentPopup()
			}),
		),
	)
}

func RawInfoModal(id, title, description, extra string, isOpenAsar bool) g.Widget {
	titleColor := ThemeAccentPurple
	switch id {
	case "#patched", "#openasar-patched":
		titleColor = ThemeAccentGreen
	case "#scuffed-install", "#insufficient-permissions", "#invalid-custom-location":
		titleColor = ThemeAccentYellow
	}

	return styledModal(
		id,
		g.Style().
			SetColor(g.StyleColorText, titleColor).
			SetFont(fontSemiBold(30)).
			To(g.Label(title)),
		g.Dummy(0, scaled(8)),
		g.Style().
			SetColor(g.StyleColorText, ThemeFg0).
			SetFont(fontRegular(20)).
			To(g.Label(description)),
		&CondWidget{extra != "", func() g.Widget {
			return g.Column(
				g.Dummy(0, scaled(10)),
				g.Style().
					SetColor(g.StyleColorChildBg, ThemeBg0).
					SetColor(g.StyleColorBorder, ThemeFg0Muted2).
					SetStyle(g.StyleVarWindowPadding, scaled(12), scaled(10)).
					SetStyleFloat(g.StyleVarChildRounding, scaled(10)).
					SetStyleFloat(g.StyleVarChildBorderSize, 1).
					To(
						g.Child().
							Border(true).
							Size(scaled(580), scaled(110)).
							Layout(
								g.Style().
									SetColor(g.StyleColorText, ThemeFg1).
									SetFont(fontRegular(18)).
									To(g.Label(extra).Wrapped(true)),
							),
					),
			)
		}, nil},
		&CondWidget{id == "#scuffed-install", func() g.Widget {
			return g.Column(
				g.Dummy(0, scaled(12)),
				g.Style().SetFont(fontSemiBold(20)).To(
					accentButton("Take me there!", ThemeAccentBlue, scaled(210), scaled(40), false, "", func() {
						// this issue only exists on windows so using Windows specific path is oki
						username := os.Getenv("USERNAME")
						programData := os.Getenv("PROGRAMDATA")
						g.OpenURL("file://" + path.Join(programData, username))
					}),
				),
			)
		}, nil},
		&CondWidget{id == "#insufficient-permissions", func() g.Widget {
			return g.Column(
				g.Dummy(0, scaled(12)),
				g.Style().SetFont(fontSemiBold(20)).To(
					accentButton("Open Settings", ThemeAccentBlue, scaled(210), scaled(40), false, "", func() {
						// "App Management" permissions doesnt exist on Monterey, just use full disk access for now
						g.OpenURL("x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles")
					}),
				),
			)
		}, nil},
		g.Dummy(0, scaled(18)),
		&CondWidget{isOpenAsar,
			func() g.Widget {
				return g.Style().SetFont(fontSemiBold(20)).To(
					g.Row(
						accentButton("Accept", ThemeAccentPurple, scaled(125), scaled(40), false, "", func() {
							acceptedOpenAsar = true
							g.CloseCurrentPopup()
						}),
						secondaryButton("Cancel", scaled(125), scaled(40), false, "", func() {
							g.CloseCurrentPopup()
						}),
					),
				)
			},
			func() g.Widget {
				return g.Style().SetFont(fontSemiBold(20)).To(
					accentButton("Ok", ThemeAccentPurple, scaled(120), scaled(40), false, "", func() {
						g.CloseCurrentPopup()
					}),
				)
			},
		},
	)
}

func PostActionModal(id, title, shortDesc, fallbackDesc string) g.Widget {
	titleColor := ThemeAccentPurple
	if id == "#patched" || id == "#openasar-patched" {
		titleColor = ThemeAccentGreen
	}

	return styledModal(
		id,
		g.Custom(func() {
			description := fallbackDesc
			if relaunchTarget != nil {
				description = shortDesc + "\n\n" + branchDisplayName(relaunchTarget.branch) + " is currently running. Would you like to relaunch it now?"
			} else if relaunchNote != "" {
				description = shortDesc + "\n\n" + relaunchNote
			}

			var buttons g.Widget
			if relaunchTarget != nil {
				target := relaunchTarget
				snapshot := relaunchSnapshot
				buttons = g.Style().SetFont(fontSemiBold(20)).To(
					g.Row(
						accentButton("Relaunch Now", ThemeAccentPurple, scaled(175), scaled(40), false, "", func() {
							relaunchTarget = nil
							relaunchSnapshot = nil
							if err := target.Relaunch(snapshot); err != nil {
								relaunchErr = err
							}
							g.CloseCurrentPopup()
						}),
						secondaryButton("Not Now", scaled(125), scaled(40), false, "", func() {
							relaunchTarget = nil
							relaunchSnapshot = nil
							g.CloseCurrentPopup()
						}),
					),
				)
			} else {
				buttons = g.Style().SetFont(fontSemiBold(20)).To(
					accentButton("Ok", ThemeAccentPurple, scaled(120), scaled(40), false, "", func() {
						g.CloseCurrentPopup()
					}),
				)
			}

			g.Layout{
				g.Style().
					SetColor(g.StyleColorText, titleColor).
					SetFont(fontSemiBold(30)).
					To(g.Label(title)),
				g.Dummy(0, scaled(8)),
				g.Style().
					SetColor(g.StyleColorText, ThemeFg0).
					SetFont(fontRegular(20)).
					To(g.Label(description)),
				g.Dummy(0, scaled(18)),
				buttons,
			}.Build()
		}),
	)
}

func UpdateModal() g.Widget {
	return styledModal(
		"#update-prompt",
		g.Style().
			SetColor(g.StyleColorText, ThemeAccentPurple).
			SetFont(fontSemiBold(30)).
			To(g.Label("Your Installer is outdated!")),
		g.Dummy(0, scaled(8)),
		g.Style().
			SetColor(g.StyleColorText, ThemeFg0).
			SetFont(fontRegular(20)).
			To(
				g.Label(
					"Would you like to update now?\n\n"+
						"Once you press Update Now, the new installer will automatically be downloaded.\n"+
						"The installer will temporarily seem unresponsive. Just wait!\n"+
						"Once the update is done, the Installer will automatically reopen.\n\n"+
						"On macOS, auto-updates are not supported, so the download will open in your browser.",
				),
			),
		g.Dummy(0, scaled(18)),
		g.Style().SetFont(fontSemiBold(20)).To(
			g.Row(
				accentButton("Update Now", ThemeAccentPurple, scaled(160), scaled(40), false, "", func() {
					if runtime.GOOS == "darwin" {
						g.CloseCurrentPopup()
						g.OpenURL(GetInstallerDownloadLink())
						return
					}

					err := UpdateSelf()
					g.CloseCurrentPopup()

					if err != nil {
						ShowModal("Failed to update self!", "Please manually download the latest Installer.", err.Error())
					} else {
						if err = RelaunchSelf(); err != nil {
							ShowModal("Failed to restart self!", "Please manually restart the Installer.", err.Error())
						}
					}
				}),
				secondaryButton("Later", scaled(125), scaled(40), false, "", func() {
					g.CloseCurrentPopup()
				}),
			),
		),
	)
}

func ShowModal(title, desc, extra string) {
	modalTitle = title
	modalMessage = desc
	modalExtra = extra
	modalId++
	g.OpenPopup("#modal" + strconv.Itoa(modalId))
}

func ShowErrorModal(title, extra, supportLink string) {
	modalTitle = title
	modalExtra = extra
	modalSupportLink = supportLink
	modalId++
	g.OpenPopup("#error-modal" + strconv.Itoa(modalId))
}

func renderHeaderMetadata() g.Widget {
	metaFont := fontRegular(18)
	metaSemi := fontSemiBold(18)
	padX := scaled(4)
	padY := scaled(2)

	var items []alignedRowItem
	hasInstallerMeta := false

	hasTag := buildinfo.InstallerTag != "" && buildinfo.InstallerTag != buildinfo.VersionUnknown
	hasHash := buildinfo.InstallerGitHash != "" && buildinfo.InstallerGitHash != buildinfo.VersionUnknown

	if hasTag {
		items = append(items, rowAlignedLabel("Installer "+buildinfo.InstallerTag, ThemeGrey2, metaFont))
		hasInstallerMeta = true
	}
	if hasHash {
		if !hasTag {
			items = append(items, rowAlignedLabel("Installer", ThemeGrey2, metaFont))
		}
		hash := buildinfo.InstallerGitHash
		url := installerCommitUrl(hash)
		items = append(items, rowAlignedLink(hash, "Open Installer commit "+hash+" on GitHub", metaSemi, padX, padY, func() {
			g.OpenURL(url)
		}))
		hasInstallerMeta = true
	}

	var buildItems []alignedRowItem
	switch {
	case IsDevInstall:
		buildItems = append(buildItems, rowAlignedLabel("Dev Install", ThemeAccentYellow, metaSemi))
	case GithubError != nil:
		buildItems = append(buildItems, rowAlignedLabel("Offline / Release fetch failed", ThemeAccentRed, metaSemi))
	case !githubDone.Load():
		buildItems = append(buildItems, rowAlignedLabel("Checking latest Vencord build...", ThemeGrey1, metaFont))
	case LatestHash != "" && LatestHash != "Unknown":
		hash := LatestHash
		url := vencordCommitUrl(hash)
		buildItems = append(buildItems,
			rowAlignedLabel("Latest Vencord:", ThemeGrey2, metaFont),
			rowAlignedLink(hash, "Open Vencord commit "+hash+" on GitHub", metaSemi, padX, padY, func() {
				g.OpenURL(url)
			}),
		)
		if InstalledHash == LatestHash {
			buildItems = append(buildItems, rowAlignedLabel("(Up to date)", ThemeAccentGreen, metaFont))
		} else if InstalledHash != "" && InstalledHash != "None" {
			installed := InstalledHash
			installedUrl := vencordCommitUrl(installed)
			buildItems = append(buildItems,
				rowAlignedLabel("|", ThemeGrey0, metaFont),
				rowAlignedLabel("Installed:", ThemeGrey2, metaFont),
				rowAlignedLink(installed, "Open installed Vencord commit "+installed+" on GitHub", metaSemi, padX, padY, func() {
					g.OpenURL(installedUrl)
				}),
			)
		}
	}

	if hasInstallerMeta && len(buildItems) > 0 {
		items = append(items, rowAlignedLabel("|", ThemeGrey0, metaFont))
	}
	items = append(items, buildItems...)

	if len(items) == 0 {
		return g.Dummy(0, 0)
	}

	return g.Custom(func() {
		var rowH float32
		g.Style().SetFont(metaFont).To(g.Custom(func() {
			_, textH := g.CalcTextSize("Ag")
			rowH = textH + padY*2
		})).Build()
		renderAlignedRow(rowH, scaled(6), items...).Build()
	})
}

func renderHeader() g.Widget {
	return g.Custom(func() {
		iconSize := scaled(60)
		startPos := imgui.CursorPos()
		startX := snap(startPos.X)
		startY := snap(startPos.Y)

		if appIcon != nil {
			var titleH, metaH float32
			g.Style().SetFont(fontSemiBold(40)).To(g.Custom(func() {
				_, titleH = g.CalcTextSize("Vencord Installer")
			})).Build()
			g.Style().SetFont(fontRegular(18)).To(g.Custom(func() {
				_, metaH = g.CalcTextSize("Ag")
			})).Build()

			stackH := titleH + scaled(8) + metaH + scaled(4)
			iconY := startY
			if stackH > iconSize {
				iconY = snap(startY + (stackH-iconSize)/2)
			}
			imgui.SetCursorPos(imgui.Vec2{X: startX, Y: iconY})
			g.ImageWithRgba(appIcon).Size(iconSize, iconSize).Build()

			imgui.SetCursorPos(imgui.Vec2{X: startX + iconSize + scaled(16), Y: startY})
		}

		g.Column(
			g.Style().
				SetColor(g.StyleColorText, ThemeAccentPurple).
				SetFont(fontSemiBold(40)).
				To(g.Label("Vencord Installer")),
			renderHeaderMetadata(),
		).Build()
	})
}

func renderInstallListCard(candidates []any, maxListHeight float32) g.Widget {
	headingFont := fontSemiBold(28)
	radioFont := fontSemiBold(20)
	badgeFont := fontSemiBold(18)
	pathFont := fontRegular(20)

	return renderAutoHeightCard(
		"InstallSectionCard",
		ThemeBgCurrentWord,
		ThemeFg0Muted2,
		1,
		scaled(14),
		scaled(22),
		scaled(18),
		scaled(180),
		maxListHeight,
		g.Custom(func() {
			startPos := imgui.CursorPos()
			startX := snap(startPos.X)
			startY := snap(startPos.Y)
			availW, _ := g.GetAvailableRegion()
			refreshW := scaled(115)
			refreshH := scaled(34)

			var titleH float32
			g.Style().
				SetColor(g.StyleColorText, ThemeFg1).
				SetFont(headingFont).
				To(g.Custom(func() {
					_, titleH = g.CalcTextSize("Select a Discord Install")
					rowH := titleH
					if refreshH > rowH {
						rowH = refreshH
					}
					imgui.SetCursorPos(imgui.Vec2{X: startX, Y: snap(startY + (rowH-titleH)/2)})
					g.Label("Select a Discord Install").Build()
				})).Build()

			rowH := titleH
			if refreshH > rowH {
				rowH = refreshH
			}
			targetX := snap(startX + availW - refreshW)
			imgui.SetCursorPos(imgui.Vec2{X: targetX, Y: snap(startY + (rowH-refreshH)/2)})
			g.Style().SetFont(fontSemiBold(18)).To(
				secondaryButton("Refresh", refreshW, refreshH, false, "Re-scan for Discord installations", refreshDiscords),
			).Build()

			imgui.SetCursorPos(imgui.Vec2{X: startX, Y: snap(startY + rowH)})
			imgui.Dummy(imgui.Vec2{X: 0, Y: 0})
		}),
		g.Dummy(0, scaled(2)),
		g.Separator(),
		g.Dummy(0, scaled(6)),

		&CondWidget{len(discords) == 0, func() g.Widget {
			s := "No Discord installs found. You first need to install Discord."
			if runtime.GOOS == "linux" {
				s += " Snap is not supported."
			}

			return &CondWidget{!showCustomLocation, func() g.Widget {
				return g.Column(
					g.Style().
						SetColor(g.StyleColorText, ThemeFg0).
						SetFont(fontRegular(24)).
						To(g.Label(s).Wrapped(true)),
					g.Dummy(0, scaled(10)),
					g.Style().SetFont(fontRegular(18)).To(
						g.Checkbox("I am an advanced user and have Discord installed at a different location", &showCustomLocation).
							OnChange(func() {
								if showCustomLocation {
									radioIdx = customChoiceIdx
								} else {
									clampRadioIdx()
								}
							}),
					),
				)
			}, nil}
		}, nil},

		g.RangeBuilder("Discords", discords, func(i int, v any) g.Widget {
			d := v.(*DiscordInstall)
			label := branchDisplayName(d.branch)

			items := []alignedRowItem{
				rowAlignedRadio(label+"##discord_"+strconv.Itoa(i), radioIdx == i, radioFont, makeRadioOnChange(i)),
			}

			if d.isPatched {
				items = append(items, rowAlignedLabel("[Vencord Installed]", ThemeAccentGreen, badgeFont))
			}
			if d.isOpenAsar != nil && *d.isOpenAsar {
				items = append(items, rowAlignedLabel("[OpenAsar]", ThemeAccentAqua, badgeFont))
			}
			if d.isFlatpak {
				items = append(items, rowAlignedLabel("[Flatpak]", ThemeAccentBlue, badgeFont))
			}

			items = append(items, rowAlignedLabel(d.path, ThemeGrey1, pathFont))

			return g.Custom(func() {
				var rowH float32
				g.Style().SetFont(radioFont).To(g.Custom(func() {
					_, framePadY := g.GetFramePadding()
					_, textH := g.CalcTextSize("Ag")
					rowH = textH + framePadY*2
				})).Build()
				renderAlignedRow(rowH, scaled(10), items...).Build()
			})
		}),

		&CondWidget{showCustomLocation, func() g.Widget {
			return g.Style().SetFont(radioFont).To(
				g.RadioButton("Custom Install Location", radioIdx == customChoiceIdx).
					OnChange(makeRadioOnChange(customChoiceIdx)),
			)
		}, nil},

		&CondWidget{showCustomLocation, func() g.Widget {
			return g.Custom(func() {
				availW, _ := g.GetAvailableRegion()
				g.Column(
					g.Dummy(0, scaled(6)),
					g.Style().
						SetColor(g.StyleColorFrameBg, ThemeBg0).
						SetColor(g.StyleColorFrameBgHovered, ThemeBg0).
						SetColor(g.StyleColorFrameBgActive, ThemeBg0).
						SetColor(g.StyleColorBorder, ThemeFg0Muted).
						SetStyle(g.StyleVarFramePadding, scaled(14), scaled(12)).
						SetStyleFloat(g.StyleVarFrameBorderSize, 1).
						SetStyleFloat(g.StyleVarFrameRounding, scaled(10)).
						SetFont(fontRegular(20)).
						To(
							g.InputText(&customDir).Hint("Enter path to your Discord install folder (Tab to autocomplete)").
								Size(availW).
								Flags(g.InputTextFlagsCallbackCompletion).
								OnChange(onCustomInputChanged).
								// this library has its own autocomplete but it's broken
								Callback(
									func(data imgui.InputTextCallbackData) int32 {
										if len(candidates) == 0 {
											return 0
										}
										// just wrap around
										if autoCompleteIdx >= len(candidates) {
											autoCompleteIdx = 0
										}

										// used by change handler
										didAutoComplete = true

										start := len(customDir)
										// Delete previous auto complete
										if lastAutoComplete != "" {
											start -= len(lastAutoComplete)
											data.DeleteBytes(start, len(lastAutoComplete))
										} else if autoCompleteFile != "" { // delete partial input
											start -= len(autoCompleteFile)
											data.DeleteBytes(start, len(autoCompleteFile))
										}

										// Insert auto complete
										lastAutoComplete = candidates[autoCompleteIdx].(string)
										data.InsertBytes(start, []byte(lastAutoComplete))
										autoCompleteIdx++

										return 0
									},
								),
						),
					&CondWidget{len(candidates) > 0, func() g.Widget {
						return g.Column(
							g.Dummy(0, scaled(6)),
							g.Style().
								SetColor(g.StyleColorChildBg, ThemeBg0).
								SetColor(g.StyleColorBorder, ThemeFg0Muted2).
								SetStyle(g.StyleVarWindowPadding, scaled(10), scaled(8)).
								SetStyleFloat(g.StyleVarChildRounding, scaled(8)).
								SetStyleFloat(g.StyleVarChildBorderSize, 1).
								SetFont(fontRegular(18)).
								To(
									g.Child().
										ID("AutoCompleteList").
										Border(true).
										Size(g.Auto, scaled(120)).
										Layout(
											g.RangeBuilder("AutoComplete", candidates, func(i int, v any) g.Widget {
												entry := v.(string)
												return g.Selectable(entry).OnClick(func() {
													customDir = path.Join(autoCompleteDir, entry)
													onCustomInputChanged()
												})
											}),
										),
								),
						)
					}, nil},
				).Build()
			})
		}, nil},
	)
}

func renderActionButtons(currentDiscord *DiscordInstall, isOpenAsar bool) g.Widget {
	return g.Custom(func() {
		availW, _ := g.GetAvailableRegion()
		spacingX, _ := g.GetItemSpacing()
		btnW := snap((availW - spacingX*3) / 4)
		if btnW < scaled(130) {
			btnW = scaled(130)
		}
		btnH := scaled(52)

		noSelection := !hasValidSelection()
		waitingGithub := !IsDevInstall && !githubDone.Load()
		installDisabled := GithubError != nil || waitingGithub || noSelection
		actionDisabled := noSelection

		installTip := "Patch the selected Discord install with Vencord"
		if GithubError != nil {
			installTip = "Unavailable: failed to fetch Vencord release info from GitHub"
		} else if waitingGithub {
			installTip = "Waiting for latest Vencord release info from GitHub..."
		} else if noSelection {
			installTip = "Select a Discord installation first"
		}

		repairTip := "Re-download the latest Vencord build and repair the selected install"
		if installDisabled {
			repairTip = installTip
		}

		openAsarLabel := Ternary(isOpenAsar, "Uninstall OpenAsar", Ternary(currentDiscord != nil, "Install OpenAsar", "(Un-)Install OpenAsar"))
		openAsarColor := Ternary(isOpenAsar, ThemeAccentRed, ThemeAccentAqua)
		openAsarTip := Ternary(isOpenAsar, "Restore Discord's original app.asar", "Install OpenAsar for faster startup & performance")

		g.Style().SetFont(fontSemiBold(20)).To(
			g.Row(
				accentButton("Install", ThemeAccentGreen, btnW, btnH, installDisabled, installTip, handlePatch),
				accentButton("Reinstall / Repair", ThemeAccentBlue, btnW, btnH, installDisabled, repairTip, func() {
					if IsDevInstall {
						handlePatch()
					} else {
						err := InstallLatestBuilds()
						if err == nil {
							handlePatch()
						}
					}
				}),
				accentButton("Uninstall", ThemeAccentRed, btnW, btnH, actionDisabled, "Remove Vencord from the selected Discord install", handleUnpatch),
				accentButton(openAsarLabel, openAsarColor, btnW, btnH, actionDisabled, openAsarTip, handleOpenAsar),
			),
		).Build()
	})
}

func renderFooter() g.Widget {
	footerFont := fontRegular(18)
	padX := scaled(6)
	padY := scaled(3)

	items := []alignedRowItem{
		rowAlignedLink("vencord.dev", "https://vencord.dev", footerFont, padX, padY, func() {
			g.OpenURL("https://vencord.dev")
		}),
		rowAlignedLabel("|", ThemeGrey0, footerFont),
		rowAlignedLink("Support & FAQ", "https://vencord.dev/support", footerFont, padX, padY, func() {
			g.OpenURL("https://vencord.dev/support")
		}),
		rowAlignedLabel("|", ThemeGrey0, footerFont),
		rowAlignedLink("GitHub", VencordRepoUrl, footerFont, padX, padY, func() {
			g.OpenURL(VencordRepoUrl)
		}),
	}

	if zoomScale != defaultZoomScale {
		pct := int(math.Round(float64(zoomScale) * 100))
		items = append(items,
			rowAlignedLabel("|", ThemeGrey0, footerFont),
			rowAlignedLink(fmt.Sprintf("Zoom %d%% (Reset)", pct), "Click or press Ctrl+0 to reset zoom", footerFont, padX, padY, func() {
				performZoomAction(zoomReset)
			}),
		)
	}

	return g.Custom(func() {
		var rowH float32
		g.Style().SetFont(footerFont).To(g.Custom(func() {
			_, textH := g.CalcTextSize("Ag")
			rowH = textH + padY*2
		})).Build()
		renderAlignedRow(rowH, scaled(6), items...).Build()
	})
}

func renderInstaller() g.Widget {
	candidates := makeAutoComplete()
	clampRadioIdx()

	currentDiscord := getSelectedDiscord()
	isOpenAsar := currentDiscord != nil && currentDiscord.IsOpenAsar()

	if CanUpdateSelf() && !showedUpdatePrompt {
		showedUpdatePrompt = true
		g.OpenPopup("#update-prompt")
	}

	if relaunchErr != nil {
		err := relaunchErr
		relaunchErr = nil
		ShowModal("Failed to relaunch Discord", "Please restart Discord manually.", err.Error())
	}

	return g.Layout{
		renderCalloutCard(
			"OfficialDownloadNoticeCard",
			ThemeAccentYellow,
			ThemeBgVisualYellow,
			"Official Download Notice",
			"GitHub and vencord.dev are the only official places to get Vencord. Any other site claiming to be us is malicious.\n"+
				"If you downloaded from any other source, delete / uninstall everything immediately, run a malware scan, and change your Discord password.",
			nil,
		),

		g.Dummy(0, scaled(16)),

		// Shrink tightly to fit installs, capping at remaining space above the buttons & footer so they never clip.
		g.Custom(func() {
			_, availH := g.GetAvailableRegion()
			reservedBottom := scaled(124) // gap + 52px buttons + spacing + footer
			maxListH := availH - reservedBottom
			if maxListH < scaled(160) {
				maxListH = scaled(160)
			}
			renderInstallListCard(candidates, maxListH).Build()
		}),

		g.Dummy(0, scaled(18)),

		renderActionButtons(currentDiscord, isOpenAsar),

		g.Dummy(0, scaled(10)),

		renderFooter(),

		PostActionModal("#patched", "Installed!", "Vencord was successfully installed!", "Vencord was successfully installed!"),
		PostActionModal("#unpatched", "Uninstalled", "Vencord has been uninstalled!", "Vencord has been uninstalled!"),
		InfoModal("#scuffed-install", "Hold On!", "You have a broken Discord Install.\n"+
			"Sometimes Discord decides to install to the wrong location for some reason!\n"+
			"You need to fix this before patching, otherwise Vencord will likely not work.\n\n"+
			"Use the below button to jump there and delete any folder called Discord or Squirrel.\n"+
			"If the folder is now empty, feel free to go back a step and delete that folder too.\n"+
			"Then see if Discord still starts. If not, reinstall it."),
		RawInfoModal("#openasar-confirm", "OpenAsar", "OpenAsar is an open-source alternative of Discord desktop's app.asar.\n"+
			"Vencord is in no way affiliated with OpenAsar.\n"+
			"You're installing OpenAsar at your own risk. If you run into issues with OpenAsar,\n"+
			"no support will be provided, join the OpenAsar Server instead!\n\n"+
			"To install OpenAsar, press Accept and click 'Install OpenAsar' again.", "", true),
		InfoModal("#insufficient-permissions", "Insufficient Permissions", "Permission denied. Please grant the installer permissions in the settings."),
		PostActionModal("#openasar-patched", "Successfully Installed OpenAsar", "OpenAsar was successfully installed!", "If Discord is still open, fully close it first. Then start it again and verify OpenAsar installed successfully!"),
		PostActionModal("#openasar-unpatched", "Successfully Uninstalled OpenAsar", "OpenAsar has been uninstalled!", "If Discord is still open, fully close it first. Then start it again and it should be back to stock!"),
		InfoModal("#invalid-custom-location", "Invalid Location", "The specified location is not a valid Discord install.\nMake sure you select the base folder.\n\nHint: Discord snap is not supported. Use flatpak or .deb."),
		InfoModalExtra("#modal"+strconv.Itoa(modalId), modalTitle, modalMessage, modalExtra),
		ErrorModal("#error-modal"+strconv.Itoa(modalId), modalTitle, modalExtra, modalSupportLink),
		UpdateModal(),
	}
}

func loop() {
	syncPixelScale()
	handleWheelZoom()
	pushedVars := pushGlobalStyleVars()

	g.SingleWindow().
		RegisterKeyboardShortcuts(
			g.WindowShortcut{Key: g.KeyUp, Callback: func() {
				if radioIdx > 0 {
					radioIdx--
				}
			}},
			g.WindowShortcut{Key: g.KeyDown, Callback: func() {
				if radioIdx < maxRadioIdx() {
					radioIdx++
				}
			}},
		).
		Layout(
			g.Style().SetFont(fontRegular(20)).To(
				renderHeader(),
				g.Dummy(0, scaled(16)),

				&CondWidget{
					GithubError != nil,
					func() g.Widget {
						return g.Column(
							renderCalloutCard(
								"GithubErrorCard",
								ThemeAccentRed,
								ThemeBgVisualRed,
								"Failed to fetch release info from GitHub",
								"Vencord build downloads are currently unavailable.",
								g.Column(
									g.Custom(func() {
										labelFont := fontRegular(20)
										linkFont := fontSemiBold(20)
										padX := scaled(3)
										padY := scaled(2)
										var rowH float32
										g.Style().SetFont(labelFont).To(g.Custom(func() {
											_, textH := g.CalcTextSize("Ag")
											rowH = textH + padY*2
										})).Build()
										renderAlignedRow(
											rowH,
											scaled(2),
											rowAlignedLabel("If this issue persists, visit", ThemeFg1, labelFont),
											rowAlignedLink("our support page", "https://vencord.dev/support", linkFont, padX, padY, func() {
												g.OpenURL("https://vencord.dev/support")
											}),
											rowAlignedLabel("for help.", ThemeFg1, labelFont),
										).Build()
									}),
									g.Dummy(0, scaled(8)),
									g.Style().SetFont(fontSemiBold(18)).To(
										accentButton("Retry", ThemeAccentRed, scaled(115), scaled(34), false, "Retry fetching release info from GitHub", retryGithubFetch),
									),
								),
							),
							g.Dummy(0, scaled(14)),
						)
					},
					nil,
				},

				&CondWidget{
					predicate:  FilesDirErr != nil,
					ifWidget:   renderFilesDirErr,
					elseWidget: renderInstaller,
				},
			),
		)

	g.PopStyleV(pushedVars)
}
