//go:build cli
/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"flag"
	"fmt"
	"os"
)

var discords []any

func isValidBranch(branch string) bool {
	switch branch {
	case "", "stable", "ptb", "canary", "auto":
		return true
	default:
		return false
	}
}

func die(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

func main() {
	InitGithubDownloader()
	discords = FindDiscords()

	var installFlag = flag.Bool("install", false, "Install Vencord on a Discord install")
	var uninstallFlag = flag.Bool("uninstall", false, "Uninstall Vencord from a Discord install")
	var installOpenAsar = flag.Bool("install-openasar", false, "Install OpenAsar on a Discord install")
	var uninstallOpenAsar = flag.Bool("uninstall-openasar", false, "Uninstall OpenAsar from a Discord install")
	var updateFlag = flag.Bool("update", false, "Update your local Vencord files")
	var locationFlag = flag.String("location", "", "Select the location of your Discord install")
	var branchFlag = flag.String("branch", "", "Select the branch of Discord you want to modify [default|stable|ptb|canary]")
	flag.Parse()

	if *locationFlag != "" && *branchFlag != "" {
		die("The 'location' and 'branch' flags are mutually exclusive.")
	}

	if !isValidBranch(*branchFlag) {
		die("The 'branch' flag must be one of the following: [auto|stable|ptb|canary]")
	}

	if *installFlag || *updateFlag {
		if !<-GithubDoneChan {
			die("Not " + Ternary(*installFlag, "installing", "updating") + " as fetching release data failed")
		}
	}

	fmt.Println("Vencord Installer cli", InstallerTag, "("+InstallerGitHash+")")

	var err error
	if *installFlag {
		_ = PromptDiscord("patch", *locationFlag, *branchFlag).patch()
	} else if *uninstallFlag {
		_ = PromptDiscord("unpatch", *locationFlag, *branchFlag).unpatch()
	} else if *updateFlag {
		_ = installLatestBuilds()
	} else if *installOpenAsar {
		discord := PromptDiscord("patch", *locationFlag, *branchFlag)
		if !discord.IsOpenAsar() {
			err = discord.InstallOpenAsar()
		} else {
			die("OpenAsar already installed")
		}
	} else if *uninstallOpenAsar {
		discord := PromptDiscord("patch", *locationFlag, *branchFlag)
		if discord.IsOpenAsar() {
			err = discord.UninstallOpenAsar()
		} else {
			die("OpenAsar not installed")
		}
	} else {
		flag.Usage()
	}

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func PromptDiscord(action, dir, branch string) *DiscordInstall {
	if branch == "default" {
		for _, b := range []string{"stable", "canary", "ptb"} {
			for _, discord := range discords {
				install := discord.(*DiscordInstall)
				if install.branch == b {
					return install
				}
			}
		}
		die("No Discord install found. Try manually specifying it with the --dir flag")
	}

	if branch != "" {
		for _, discord := range discords {
			install := discord.(*DiscordInstall)
			if install.branch == branch {
				return install
			}
		}
		die("Discord " + branch + " not found")
	}

	if dir != "" {
		if discord := ParseDiscord(dir, branch); discord != nil {
			return discord
		} else {
			die(dir + " is not a valid Discord install")
		}
	}

	fmt.Println("Please choose a Discord install to", action)

	for i, discord := range discords {
		install := discord.(*DiscordInstall)
		fmt.Printf("[%d] %s%s (%s)\n", i+1, Ternary(install.isPatched, "(PATCHED) ", ""), install.path, install.branch)
	}

	fmt.Printf("[%d] Custom Location\n", len(discords)+1)

	var choice int
	for {
		fmt.Printf("> ")
		if _, err := fmt.Scan(&choice); err != nil {
			fmt.Println("That wasn't a valid choice")
			continue
		}

		choice--
		if choice >= 0 && choice < len(discords) {
			return discords[choice].(*DiscordInstall)
		}

		if choice == len(discords) {
			var custom string
			fmt.Print("Custom Discord Install: ")
			if _, err := fmt.Scan(&custom); err == nil {
				if discord := ParseDiscord(custom, branch); discord != nil {
					return discord
				}
			}
		}

		fmt.Println("That wasn't a valid choice")
	}
}

func InstallLatestBuilds() error {
	return installLatestBuilds()
}

func HandleScuffedInstall() {
	fmt.Println("Hold On!")
	fmt.Println("You have a broken Discord Install.")
	fmt.Println("Please reinstall Discord before proceeding!")
	fmt.Println("Otherwise, Vencord will likely not work.")
}
