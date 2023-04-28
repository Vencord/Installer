//go:build gui || (!gui && !cli)
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
	g "github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
	"image"
	"image/color"
	// png decoder for icon
	_ "image/png"
	"os"
	path "path/filepath"
	"runtime"
	"strconv"
	"strings"
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

	modalId      = 0
	modalTitle   = "Oh No :("
	modalMessage = "You should never see this"

	acceptedOpenAsar bool

	win *g.MasterWindow
)

//go:embed winres/icon.png
var iconBytes []byte

func main() {
	InitGithubDownloader()
	discords = FindDiscords()

	customChoiceIdx = len(discords)

	go func() {
		<-GithubDoneChan
		g.Update()
	}()

	go func() {
		CheckSelfUpdate()
		g.Update()
	}()

	win = g.NewMasterWindow("Installateur Vencord", 1200, 800, 0)

	icon, _, err := image.Decode(bytes.NewReader(iconBytes))
	if err != nil {
		fmt.Println("Erreur lors du chargement de l'icone", err)
		fmt.Println(iconBytes, len(iconBytes))
	} else {
		win.SetIcon([]image.Image{icon})
	}
	win.Run(loop)
}

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
			ShowModal("Alors...", "Cela ne ressemble pas a une installation Discord...\nMerci de vérifier que vous avez selectionné le dossier de base, par exemple\n(blah/Discord, et pas blah/Discord/resources/app)")
		}
	} else {
		choice = discords[radioIdx].(*DiscordInstall)
	}
	return choice
}

func InstallLatestBuilds() (err error) {
	if IsDevInstall {
		return
	}

	err = installLatestBuilds()
	if err != nil {
		ShowModal("Aie! ", "Erreur lors de l'installation des derniers builds de Vencord depuis Github :\n"+err.Error())
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
	if acceptedOpenAsar || getChosenInstall().IsOpenAsar() {
		handleOpenAsarConfirmed()
		return
	}

	g.OpenPopup("#openasar-confirm")
}

func handleOpenAsarConfirmed() {
	choice := getChosenInstall()
	if choice != nil {
		if choice.IsOpenAsar() {
			if err := choice.UninstallOpenAsar(); err != nil {
				handleErr(err, "désinstaller OpenAsar de")
			} else {
				g.OpenPopup("#openasar-unpatched")
				g.Update()
			}
		} else {
			if err := choice.InstallOpenAsar(); err != nil {
				handleErr(err, "installer OpenAsar dans")
			} else {
				g.OpenPopup("#openasar-patched")
				g.Update()
			}
		}
	}
}

func handleErr(err error, action string) {
	if errors.Is(err, os.ErrPermission) {
		switch os := runtime.GOOS; os {
		case "windows":
			err = errors.New("Permission refusée. Essayez de fermer Discord depuis le gestionnaire des tâches et de réessayer.")
		case "darwin":
			err = errors.New("Permission refusée. Merci de donner les permissions nécessaires à l'application, en autorisant l'accès complet au disque dans les paramètres de votre système (page sécuritée).")
		default:
			err = errors.New("Permission refusée. Essayez de lancer l'application en tant qu'administrateur.")
	}

	ShowModal("Erreur lors de "+action+" de cette installation", err.Error())
}

func HandleScuffedInstall() {
	g.OpenPopup("#scuffed-install")
}

func (di *DiscordInstall) Patch() {
	if CheckScuffedInstall() {
		return
	}
	if err := di.patch(); err != nil {
		handleErr(err, "patch")
	} else {
		g.OpenPopup("#patched")
	}
}

func (di *DiscordInstall) Unpatch() {
	if err := di.unpatch(); err != nil {
		handleErr(err, "unpatch")
	} else {
		g.OpenPopup("#unpatched")
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
					g.Label("Erreur lors de la création de : "+FilesDirErr.Error()),
					g.Label("Merci de régler le problème et de réessayer."),
				),
			),
	}
}

func Tooltip(label string) g.Widget {
	return g.Style().
		SetStyle(g.StyleVarWindowPadding, 10, 8).
		SetStyleFloat(g.StyleVarWindowRounding, 8).
		To(
			g.Tooltip(label),
		)
}

func InfoModal(id, title, description string) g.Widget {
	return RawInfoModal(id, title, description, false)
}

func RawInfoModal(id, title, description string, isOpenAsar bool) g.Widget {
	isDynamic := strings.HasPrefix(id, "#modal")
	return g.Style().
		SetStyle(g.StyleVarWindowPadding, 30, 30).
		SetStyleFloat(g.StyleVarWindowRounding, 12).
		To(
			g.PopupModal(id).
				Flags(g.WindowFlagsNoTitleBar | Ternary(isDynamic, g.WindowFlagsAlwaysAutoResize, 0)).
				Layout(
					g.Align(g.AlignCenter).To(
						g.Style().SetFontSize(30).To(
							g.Label(title),
						),
						g.Style().SetFontSize(20).To(
							g.Label(description).Wrapped(isDynamic),
						),
						&CondWidget{id == "#scuffed-install", func() g.Widget {
							return g.Column(
								g.Dummy(0, 10),
								g.Button("Take me there!").OnClick(func() {
									// this issue only exists on windows so using Windows specific path is oki
									username := os.Getenv("USERNAME")
									programData := os.Getenv("PROGRAMDATA")
									g.OpenURL("file://" + path.Join(programData, username))
								}).Size(200, 30),
							)
						}, nil},
						g.Dummy(0, 20),
						&CondWidget{isOpenAsar,
							func() g.Widget {
								return g.Row(
									g.Button("Accepter").
										OnClick(func() {
											acceptedOpenAsar = true
											g.CloseCurrentPopup()
										}).
										Size(100, 30),
									g.Button("Annuler").
										OnClick(func() {
											g.CloseCurrentPopup()
										}).
										Size(100, 30),
								)
							},
							func() g.Widget {
								return g.Button("Ok").
									OnClick(func() {
										g.CloseCurrentPopup()
									}).
									Size(100, 30)
							},
						},
					),
				),
		)
}

func ShowModal(title, desc string) {
	modalTitle = title
	modalMessage = desc
	modalId++
	g.OpenPopup("#modal" + strconv.Itoa(modalId))
}

func renderInstaller() g.Widget {
	candidates := makeAutoComplete()
	wi, _ := win.GetSize()
	w := float32(wi) - 96

	var currentDiscord *DiscordInstall
	if radioIdx != customChoiceIdx {
		currentDiscord = discords[radioIdx].(*DiscordInstall)
	}
	var isOpenAsar = currentDiscord != nil && currentDiscord.IsOpenAsar()

	layout := g.Layout{
		g.Dummy(0, 20),
		g.Separator(),
		g.Dummy(0, 5),

		g.Style().SetFontSize(30).To(
			g.Label("Merci de sélectionner votre installation de Discord"),
		),

		g.Style().SetFontSize(20).To(
			g.RangeBuilder("Discords", discords, func(i int, v any) g.Widget {
				d := v.(*DiscordInstall)
				text := d.path + " (" + d.branch + ")"
				if d.isPatched {
					text = "[PATCHÉE] " + text
				}
				return g.RadioButton(text, radioIdx == i).
					OnChange(makeRadioOnChange(i))
			}),

			g.RadioButton("Emplacement personnalisé", radioIdx == customChoiceIdx).
				OnChange(makeRadioOnChange(customChoiceIdx)),
		),

		g.Dummy(0, 5),
		g.Style().
			SetStyle(g.StyleVarFramePadding, 16, 16).
			SetFontSize(20).
			To(
				g.InputText(&customDir).Hint("L'emplacement personnalisé").
					Size(w - 16).
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
						g.Button("Installer").
							OnClick(handlePatch).
							Size((w-40)/4, 50),
						Tooltip("Patcher l'installation de Discord sélectionnée"),
					),
				g.Style().
					SetColor(g.StyleColorButton, Ternary(isOpenAsar, DiscordRed, DiscordGreen)).
					To(
						g.Button(Ternary(isOpenAsar, "Désinstaller OpenAsar", Ternary(currentDiscord != nil, "Installer OpenAsar", "(Dé-)Install OpenAsar"))).
							OnClick(handleOpenAsar).
							Size((w-40)/4, 50),
						Tooltip("Modifier OpenAsar"),
					),
				g.Style().
					SetColor(g.StyleColorButton, DiscordRed).
					To(
						g.Button("Désinstaller").
							OnClick(handleUnpatch).
							Size((w-40)/4, 50),
						Tooltip("Désinstaller Vencord de l'installation de Discord sélectionnée"),
					),
				g.Style().
					SetColor(g.StyleColorButton, DiscordBlue).
					SetDisabled(IsDevInstall || GithubError != nil).
					To(
						g.Button(Ternary(GithubError == nil && LatestHash == InstalledHash, "Re-Télécharger Vencord", "Mise à jour")).
							OnClick(func() {
								if err := InstallLatestBuilds(); err == nil {
									g.OpenPopup("téléchargé")
								}
							}).
							Size((w-40)/4, 50),
						Tooltip("Mettre à jour Vencord"),
					),
			),
		),

		InfoModal("#downloaded", "Téléchargement réussi", "Vencord a été téléchargé avec succès !"),
		InfoModal("#patched", "Patch réussi ! ", "Merci de quitter Discord depuis la barre des tâches\n"+
			"Ensuite, relancez-le pour voir les changements, en allant dans Paramètres depuis les paramètres Discord, et en cherchant Vencord"),
		InfoModal("#unpatched", "Dé-Patch réussi !", "Merci de quitter Discord depuis la barre des tâches\n"+
		InfoModal("#scuffed-install", "Aie ! ", "Votre installation Discord est problématique...\n"+
			"Parfois Discord décide de s'installer dans le mauvais dossier sans raison...\n"+
			"Avant de poursuivre, vous devez régler ce problème, sans quoi Vencord ne marchera certainement pas...\n\n"+
			"Cliquez sur le bouton ci dessous pour vous y rendre, et supprimez le dossier Discord et/ou Squirrel.\n"+
			"Si le dossier est maintenant vide, supprimez complétement le dossier.\n"+
			"Vérifiez que Discord ne s'éxécute pas, et réinstallez-le".),
		RawInfoModal("#openasar-confirm", "OpenAsar", "OpenAsar est une alternative open-source au Luncher Discord app.asar.\n"+
			"Vencord est en aucun cas affilié avec OpenAsar.\n"+
			"Vous installer OpenAsar à vos propres risques \n"+
			"Si vous rencontrez des problèmes avec OpenAsar, rendez vous sur leur serveur pour demandez de l'aide!\n\n"+
			"Pour installer OpenAsar, cliquez sur Acceptez, puis sur 'Install OpenAsar' again.", true),
		InfoModal("#openasar-patched", "OpenAsar installé ! ", "Redémarrez Discord depuis le gestionnaire des tâches pour voir les changements"),
		InfoModal("#openasar-unpatched", "OpenAsar désinstallé ! ", "Redémarrez Discord depuis le gestionnaire des tâches"),
		InfoModal("#modal"+strconv.Itoa(modalId), modalTitle, modalMessage),
	}

	return layout
}

func renderErrorCard(col color.Color, message string) g.Widget {
	return g.Style().
		SetColor(g.StyleColorChildBg, col).
		SetStyleFloat(g.StyleVarAlpha, 0.9).
		SetStyle(g.StyleVarWindowPadding, 10, 10).
		SetStyleFloat(g.StyleVarChildRounding, 5).
		To(
			g.Child().
				Size(g.Auto, 40).
				Layout(
					g.Row(
						g.Style().SetColor(g.StyleColorText, color.Black).To(
							g.Markdown(&message),
						),
					),
				),
		)
}

func loop() {
	g.PushWindowPadding(48, 48)

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
					g.Label("Installateur Vencord"),
				),
			),

			g.Dummy(0, 20),

			g.Style().SetFontSize(20).To(
				g.Row(
					g.Label(Ternary(IsDevInstall, "Dev Install: ", "Les fichiers vont être installés ici: ")+FilesDir),
					g.Style().
						SetColor(g.StyleColorButton, DiscordBlue).
						SetStyle(g.StyleVarFramePadding, 4, 4).
						To(
							g.Button("Ouvrir le dossier").OnClick(func() {
								g.OpenURL("file://" + FilesDir)
							}),
						),
				),
				&CondWidget{!IsDevInstall, func() g.Widget {
					return g.Label("Pour changer l'emplacement, modifiez la variable d'environnement 'VENCORD_USER_DATA_DIR' et redémarrez l'installateur").Wrapped(true)
				}, nil},
				g.Dummy(0, 10),
				g.Label("Emplacement de l'installation: "+InstallerTag+" ("+InstallerGitHash+")"+Ternary(IsInstallerOutdated, " - PLUS À JOUR", "")),
				g.Label("Version Locale de Vencord "+InstalledHash),
				&CondWidget{
					GithubError == nil,
					func() g.Widget {
						if IsDevInstall {
							return g.Label("Pas de mise à jour pour les installations de développement")
						}
						return g.Label("Dernière Version de Vencord " + LatestHash)
					}, func() g.Widget {
						return renderErrorCard(DiscordRed, "Erreur lors de la récupération des donnéees depuis Github "+GithubError.Error())
					},
				},
				&CondWidget{
					IsInstallerOutdated,
					func() g.Widget {
						return renderErrorCard(DiscordYellow, "Cet installateur n'est pas à jour! "+GetInstallerDownloadMarkdown())
					},
					nil,
				},
			),

			&CondWidget{
				predicate:  FilesDirErr != nil,
				ifWidget:   renderFilesDirErr,
				elseWidget: renderInstaller,
			},
		)

	g.PopStyle()
}
