//go:build cli
/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"flag"
	"fmt"
)

var discords []any

func main() {
	InitGithubDownloader()
	discords = FindDiscords()

	var installFlag = flag.Bool("install", false, "Installer Vencord pour une installation Discord")
	var uninstallFlag = flag.Bool("uninstall", false, "Désinstaller Vencord pour une installation Discord")
	var installOpenAsar = flag.Bool("install-openasar", false, "Installer OpenAsar pour une installation Discord")
	var uninstallOpenAsar = flag.Bool("uninstall-openasar", false, "Désinstaller OpenAsar pour une installation Discord")
	var updateFlag = flag.Bool("update", false, "Mettre à jour vos fichiers locaux Vencord")
	flag.Parse()

	if *installFlag || *updateFlag {
		if !<-GithubDoneChan {
			fmt.Println("Not", Ternary(*installFlag, "installing", "updating"), "as fetching release data failed")
			return
		}
	}

	fmt.Println("Vencord Installer cli", InstallerTag, "("+InstallerGitHash+")")

	var err error
	if *installFlag {
		_ = PromptDiscord("patch").patch()
	} else if *uninstallFlag {
		_ = PromptDiscord("unpatch").unpatch()
	} else if *updateFlag {
		_ = installLatestBuilds()
	} else if *installOpenAsar {
		discord := PromptDiscord("patch")
		if !discord.IsOpenAsar() {
			err = discord.InstallOpenAsar()
		} else {
			err = errors.New("OpenAsar déjà installé")
		}
	} else if *uninstallOpenAsar {
		discord := PromptDiscord("patch")
		if discord.IsOpenAsar() {
			err = discord.UninstallOpenAsar()
		} else {
			err = errors.New("OpenAsar n'est pas installé")
		}
	} else {
		flag.Usage()
	}

	if err != nil {
		fmt.Println(err)
	}
}

func PromptDiscord(action string) *DiscordInstall {
	fmt.Println("Merci de choisir l'installation Discord à patcher", action)
	for i, discord := range discords {
		install := discord.(*DiscordInstall)
		fmt.Printf("[%d] %s%s (%s)\n", i+1, Ternary(install.isPatched, "(PATCHED) ", ""), install.path, install.branch)
	}
	fmt.Printf("[%d] Emplacement personnalisée\n", len(discords)+1)

	var choice int
	for {
		fmt.Printf("> ")
		if _, err := fmt.Scan(&choice); err != nil {
			fmt.Println("Choix non valide")
			continue
		}

		choice--
		if choice >= 0 && choice < len(discords) {
			return discords[choice].(*DiscordInstall)
		}

		if choice == len(discords) {
			var custom string
			fmt.Print("Installation personnalisée: ")
			if _, err := fmt.Scan(&custom); err == nil {
				if discord := ParseDiscord(custom, ""); discord != nil {
					return discord
				}
			}
		}

		fmt.Println("Choix non valide")
	}
}

func InstallLatestBuilds() error {
	return installLatestBuilds()
}

func HandleScuffedInstall() {
	fmt.Println("Attention!")
	fmt.Println("Votre installation Discord est corrompue\nMerci de réinstaller Discord avant de re-essayer!\nSinon, Vencord ne fonctionnera pas correctement!")
}
