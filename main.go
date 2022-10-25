/*
 * This part is file of VencordInstaller
 * Copyright (c) 2022 Vendicated
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"os"
	"path"
	"strings"

	g "github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
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

	win *g.MasterWindow
)

type CondWidget struct {
	predicate  bool
	ifWidget   func() g.Widget
	elseWidget func() g.Widget
}

func (w *CondWidget) Build() {
	if w.predicate {
		w.ifWidget().Build()
	} else if w.elseWidget != nil {
		w.elseWidget().Build()
	}
}

func getChosenInstall() *DiscordInstall {
	var choice *DiscordInstall
	if radioIdx == customChoiceIdx {
		choice = ParseDiscord(customDir, "")
		if choice == nil {
			g.Msgbox("Uh Oh!", "That doesn't seem to be a Discord install. Please make sure you select the base folder (blah/Discord, not blah/Discord/resources/app)").
				Buttons(g.MsgboxButtonsOk)
		}
	} else {
		choice = discords[radioIdx].(*DiscordInstall)
	}
	return choice
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

func onCustomInputChanged() {
	p := customDir
	if len(p) != 0 {
		// Select the custom option for people
		radioIdx = customChoiceIdx
	}

	dir := path.Dir(p)

	isNewDir := strings.HasSuffix(p, "/")
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

func renderFilesDirErr() g.Widget {
	return g.Layout{
		g.Dummy(0, 50),
		g.Style().
			SetColor(g.StyleColorText, DiscordRed).
			SetFontSize(30).
			To(
				g.Align(g.AlignCenter).To(
					g.Label("Error: Failed to create: "+FilesDirErr.Error()),
					g.Label("Resolve this error, then restart me!"),
				),
			),
	}
}

func renderInstaller() g.Widget {
	candidates := makeAutoComplete()
	wi, _ := win.GetSize()
	w := float32(wi)

	return g.Layout{
		g.Dummy(0, 20),
		g.Separator(),
		g.Dummy(0, 5),

		g.Style().SetFontSize(30).To(
			g.Label("Please select an install to patch"),
		),

		g.Style().SetFontSize(20).To(
			g.RangeBuilder("Discords", discords, func(i int, v any) g.Widget {
				d := v.(*DiscordInstall)
				text := d.path + " (" + d.branch + ")"
				if d.isPatched {
					text = "[PATCHED] " + text
				}
				return g.RadioButton(text, radioIdx == i).
					OnChange(makeRadioOnChange(i))
			}),

			g.RadioButton("Custom Install Location", radioIdx == customChoiceIdx).
				OnChange(makeRadioOnChange(customChoiceIdx)),
		),

		g.Dummy(0, 5),
		g.Style().
			SetStyle(g.StyleVarFramePadding, 16, 16).
			SetFontSize(20).
			To(
				g.InputText(&customDir).Hint("The custom location").
					Size(w).
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
		g.RangeBuilder("AutoComplete", candidates, func(i int, v any) g.Widget {
			dir := v.(string)
			return g.Label(dir)
		}),

		g.Dummy(0, 20),

		g.Style().SetFontSize(20).To(
			g.Row(
				g.Style().
					SetColor(g.StyleColorButton, DiscordGreen).
					SetDisabled(GithubError != nil).
					To(
						g.Button("Patch").
							OnClick(handlePatch).
							Size(w/3, 50),
						g.Tooltip("Patch the selected Discord Install"),
					),
				g.Style().
					SetColor(g.StyleColorButton, DiscordRed).
					SetDisabled(GithubError != nil).
					To(
						g.Button("Unpatch").
							OnClick(handleUnpatch).
							Size(w/3, 50),
						g.Tooltip("Unpatch the selected Discord Install"),
					),
				g.Style().
					SetColor(g.StyleColorButton, DiscordBlue).
					SetDisabled(GithubError != nil).
					To(
						g.Button(Ternary(GithubError == nil && LatestHash == InstalledHash, "Re-Download Vencord", "Update")).
							OnClick(func() {
								_ = InstallLatestBuilds()
							}).
							Size(w/3, 50),
						g.Tooltip("Update your local Vencord files"),
					),
			),
		),

		g.PrepareMsgbox(),
	}
}

func loop() {
	// Todo: Figure out how to add padding around window
	g.SingleWindow().
		RegisterKeyboardShortcuts(
			g.WindowShortcut{Key: g.KeyUp, Callback: func() {
				if radioIdx > 0 {
					radioIdx--
				}
			}},
			g.WindowShortcut{Key: g.KeyDown, Callback: func() {
				if radioIdx < customChoiceIdx {
					radioIdx++
				}
			}},
		).
		Layout(
			g.Align(g.AlignCenter).To(
				g.Style().SetFontSize(40).To(
					g.Label("Vencord Installer"),
				),
			),

			g.Dummy(0, 20),

			g.Style().SetFontSize(20).To(
				g.Row(
					g.Label("Files will be downloaded to: "+FilesDir),
					g.Style().
						SetColor(g.StyleColorButton, DiscordBlue).
						SetStyle(g.StyleVarFramePadding, 4, 4).
						To(
							g.Button("Open Directory").OnClick(func() {
								g.OpenURL("file://" + FilesDir)
							}),
						),
				),
				g.Label("To customise this location, set the environment variable 'VENCORD_USER_DATA_DIR' and restart me").Wrapped(true),
				g.Dummy(0, 10),
				g.Label("Installer Version: "+InstallerGitHash),
				g.Label("Local Vencord Version: "+InstalledHash),
				&CondWidget{
					GithubError == nil,
					func() g.Widget {
						return g.Label("Latest Vencord Version: " + LatestHash)
					}, func() g.Widget {
						return g.Style().
							SetColor(g.StyleColorText, DiscordRed).
							To(
								g.Align(g.AlignCenter).To(
									g.Label("Failed to fetch Info from GitHub: "+GithubError.Error()),
									g.Label("Resolve this error, then restart me!"),
								),
							)
					},
				},
			),

			&CondWidget{
				predicate:  FilesDirErr != nil,
				ifWidget:   renderFilesDirErr,
				elseWidget: renderInstaller,
			},
		)
}

func main() {
	// this init function depends on patcher init, so using the automatic init won't work
	InitGithubDownloader()

	discords = FindDiscords()
	customChoiceIdx = len(discords)

	win = g.NewMasterWindow("Vencord Installer", 1200, 800, g.MasterWindowFlags(0))
	win.Run(loop)
}
