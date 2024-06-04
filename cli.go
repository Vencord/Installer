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
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"os"
	"runtime"
	"strings"
	"vencordinstaller/buildinfo"
)

var discords []any
var interactive = false

func isValidBranch(branch string) bool {
	switch branch {
	case "", "stable", "ptb", "canary", "auto":
		return true
	default:
		return false
	}
}

func die(msg string) {
	Log.Error(msg)
	exitFailure()
}

func main() {
	InitGithubDownloader()
	discords = FindDiscords()

	// Used by log.go init func
	flag.Bool("debug", false, "Enable debug info")

	var helpFlag = flag.Bool("help", false, "View usage instructions")
	var versionFlag = flag.Bool("version", false, "View the program version")
	var updateSelfFlag = flag.Bool("update-self", false, "Update me to the latest version")
	var installFlag = flag.Bool("install", false, "Install Vencord")
	var updateFlag = flag.Bool("repair", false, "Repair Vencord")
	var uninstallFlag = flag.Bool("uninstall", false, "Uninstall Vencord")
	var installOpenAsarFlag = flag.Bool("install-openasar", false, "Install OpenAsar")
	var uninstallOpenAsarFlag = flag.Bool("uninstall-openasar", false, "Uninstall OpenAsar")
	var locationFlag = flag.String("location", "", "The location of the Discord install to modify")
	var branchFlag = flag.String("branch", "", "The branch of Discord to modify [auto|stable|ptb|canary]")
	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if *versionFlag {
		fmt.Println("Vencord Installer Cli", buildinfo.InstallerTag, "("+buildinfo.InstallerGitHash+")")
		fmt.Println("Copyright (C) 2023 Vendicated and Vencord contributors")
		fmt.Println("License GPLv3+: GNU GPL version 3 or later <https://gnu.org/licenses/gpl.html>.")
		return
	}

	if *updateSelfFlag {
		if !<-SelfUpdateCheckDoneChan {
			die("Can't update self because checking for updates failed")
		}
		if err := UpdateSelf(); err != nil {
			Log.Error("Failed to update self:", err)
			exitFailure()
		}
		exitSuccess()
	}

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

	install, uninstall, update, installOpenAsar, uninstallOpenAsar := *installFlag, *uninstallFlag, *updateFlag, *installOpenAsarFlag, *uninstallOpenAsarFlag
	switches := []*bool{&install, &update, &uninstall, &installOpenAsar, &uninstallOpenAsar}
	if !SliceContainsFunc(switches, func(b *bool) bool { return *b }) {
		interactive = true

		go func() {
			<-SelfUpdateCheckDoneChan
			if IsSelfOutdated {
				Log.Warn("Your installer is outdated.")
				Log.Warn("To update, select the 'Update Vencord Installer' option to update, or run with --update-self")
			}
		}()

		choices := []string{
			"Install Vencord",
			"Repair Vencord",
			"Uninstall Vencord",
			"Install OpenAsar",
			"Uninstall OpenAsar",
			"View Help Menu",
			"Update Vencord Installer",
			"Quit",
		}
		_, choice, err := (&promptui.Select{
			Label: "What would you like to do? (Press Enter to confirm)",
			Items: choices,
		}).Run()
		handlePromptError(err)

		switch choice {
		case "View Help Menu":
			flag.Usage()
			return
		case "Quit":
			return
		case "Update Vencord Installer":
			if err := UpdateSelf(); err != nil {
				Log.Error("Failed to update self:", err)
				exitFailure()
			}
			exitSuccess()
		}

		*switches[SliceIndex(choices, choice)] = true
	}

	var err error
	var errSilent error
	if install {
		errSilent = PromptDiscord("patch", *locationFlag, *branchFlag).patch()
	} else if uninstall {
		errSilent = PromptDiscord("unpatch", *locationFlag, *branchFlag).unpatch()
	} else if update {
		Log.Info("Downloading latest Vencord files...")
		err := installLatestBuilds()
		Log.Info("Done!")
		if err == nil {
			errSilent = PromptDiscord("repair", *locationFlag, *branchFlag).patch()
		}
	} else if installOpenAsar {
		discord := PromptDiscord("patch", *locationFlag, *branchFlag)
		if !discord.IsOpenAsar() {
			err = discord.InstallOpenAsar()
		} else {
			die("OpenAsar already installed")
		}
	} else if uninstallOpenAsar {
		discord := PromptDiscord("patch", *locationFlag, *branchFlag)
		if discord.IsOpenAsar() {
			err = discord.UninstallOpenAsar()
		} else {
			die("OpenAsar not installed")
		}
	}

	if err != nil {
		Log.Error(err)
		exitFailure()
	}
	if errSilent != nil {
		exitFailure()
	}

	exitSuccess()
}

func exit(status int) {
	if runtime.GOOS == "windows" && IsDoubleClickRun() && interactive {
		fmt.Print("Press Enter to exit")
		var b byte
		_, _ = fmt.Scanf("%v", &b)
	}
	os.Exit(status)
}

func exitSuccess() {
	color.HiGreen("✔ Success!")
	exit(0)
}

func exitFailure() {
	color.HiRed("❌ Failed!")
	exit(1)
}

func handlePromptError(err error) {
	if errors.Is(err, promptui.ErrInterrupt) {
		exit(0)
	}

	Log.FatalIfErr(err)
}

func PromptDiscord(action, dir, branch string) *DiscordInstall {
	if branch == "auto" {
		for _, b := range []string{"stable", "canary", "ptb"} {
			for _, discord := range discords {
				install := discord.(*DiscordInstall)
				if install.branch == b {
					return install
				}
			}
		}
		die("No Discord install found. Try manually specifying it with the --dir flag. Hint: snap is not supported")
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
			die(dir + " is not a valid Discord install. Hint: snap is not supported")
		}
	}

	items := SliceMap(discords, func(d any) string {
		install := d.(*DiscordInstall)
		//goland:noinspection GoDeprecation
		return fmt.Sprintf("%s - %s%s", strings.Title(install.branch), install.path, Ternary(install.isPatched, " [PATCHED]", ""))
	})
	items = append(items, "Custom Location")

	_, choice, err := (&promptui.Select{
		Label: "Select Discord install to " + action + " (Press Enter to confirm)",
		Items: items,
	}).Run()
	handlePromptError(err)

	if choice != "Custom Location" {
		return discords[SliceIndex(items, choice)].(*DiscordInstall)
	}

	for {
		custom, err := (&promptui.Prompt{
			Label: "Custom Discord Location",
		}).Run()
		handlePromptError(err)

		if di := ParseDiscord(custom, ""); di != nil {
			return di
		}

		Log.Error("Invalid Discord install!")
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
