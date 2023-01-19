//go:build cli

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

	var installFlag = flag.Bool("install", false, "Install Vencord on a Discord install")
	var uninstallFlag = flag.Bool("uninstall", false, "Uninstall Vencord from a Discord install")
	var installOpenAsar = flag.Bool("install-openasar", false, "Install OpenAsar on a Discord install")
	var uninstallOpenAsar = flag.Bool("uninstall-openasar", false, "Uninstall OpenAsar from a Discord install")
	var updateFlag = flag.Bool("update", false, "Update your local Vencord files")
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
			err = errors.New("OpenAsar already installed")
		}
	} else if *uninstallOpenAsar {
		discord := PromptDiscord("patch")
		if discord.IsOpenAsar() {
			err = discord.UninstallOpenAsar()
		} else {
			err = errors.New("OpenAsar not installed")
		}
	} else {
		flag.Usage()
	}

	if err != nil {
		fmt.Println(err)
	}
}

func PromptDiscord(action string) *DiscordInstall {
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
				if discord := ParseDiscord(custom, ""); discord != nil {
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
	fmt.Println("You have a broken Discord Install.\nPlease reinstall Discord before proceeding!\nOtherwise, Vencord will likely not work.")
}
