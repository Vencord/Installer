//go:build cli

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
)

var discords []any

func isAllowedClient(client string) bool {
	ignoredClients := []string{"", "default", "stable", "ptb", "canary"}
	return contains(ignoredClients, client)
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func main() {
	InitGithubDownloader()
	discords = FindDiscords()

	var installFlag = flag.Bool("install", false, "Install Vencord on a Discord install")
	var uninstallFlag = flag.Bool("uninstall", false, "Uninstall Vencord from a Discord install")
	var installOpenAsar = flag.Bool("install-openasar", false, "Install OpenAsar on a Discord install")
	var uninstallOpenAsar = flag.Bool("uninstall-openasar", false, "Uninstall OpenAsar from a Discord install")
	var updateFlag = flag.Bool("update", false, "Update your local Vencord files")
	var dir = flag.String("dir", "", "Select the location of your Discord client")
	var client = flag.String("client", "", "Select the branch of Discord you wish to modify [default|stable|ptb|canary]")
	flag.Parse()

	if *dir != "" && *client != "default" {
		log.Fatal("The 'dir' and 'client' flags are mutually exclusive.")
	}

	if !isAllowedClient(*client) {
		log.Fatal("The 'client' flag must be one of the following: [default|stable|ptb|canary]")
	}

	if *installFlag || *updateFlag {
		if !<-GithubDoneChan {
			fmt.Println("Not", Ternary(*installFlag, "installing", "updating"), "as fetching release data failed")
			return
		}
	}

	fmt.Println("Vencord Installer cli", InstallerTag, "("+InstallerGitHash+")")

	var err error
	if *installFlag {
		PromptDiscord("patch", nil, *dir, *client).patch()
	} else if *uninstallFlag {
		PromptDiscord("unpatch", nil, *dir, *client).unpatch()
	} else if *updateFlag {
		installLatestBuilds()
	} else if *installOpenAsar {
		discord := PromptDiscord("patch", nil, *dir, *client)
		if !discord.IsOpenAsar() {
			err = discord.InstallOpenAsar()
		} else {
			err = errors.New("OpenAsar already installed")
		}
	} else if *uninstallOpenAsar {
		discord := PromptDiscord("patch", nil, *dir, *client)
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

func PromptDiscord(action string, client *DiscordInstall, dir string, branch string) *DiscordInstall {

	if branch != "" {
		for _, discord := range discords {
			install := discord.(*DiscordInstall)
			if install.branch == branch {
				return install
			}
		}
	}

	fmt.Println("Please choose a Discord install to", action)

	for i, discord := range discords {
		install := discord.(*DiscordInstall)
		fmt.Printf("[%d] %s%s (%s)\n", i+1, Ternary(install.isPatched, "(PATCHED) ", ""), install.path, install.branch)
	}

	if dir != "" {
		fmt.Printf("[%d] %s\n", len(discords)+1, dir)
	} else {
		fmt.Printf("[%d] Custom Location\n", len(discords)+1)
	}

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
			if dir != "" {
				if discord := ParseDiscord(dir, branch); discord != nil {
					return discord
				}
			} else {
				var custom string
				fmt.Print("Custom Discord Install: ")
				if _, err := fmt.Scan(&custom); err == nil {
					if discord := ParseDiscord(custom, branch); discord != nil {
						return discord
					}
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
