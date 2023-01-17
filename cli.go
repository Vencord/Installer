//go:build cli

package main

import (
	"flag"
	"fmt"
)

var discords []any

func main() {
	InitGithubDownloader()
	discords = FindDiscords()

	var installFlag = flag.Bool("install", false, "Run the CLI in install mode")
	var uninstallFlag = flag.Bool("uninstall", false, "Run the CLI in uninstall mode")
	var updateFlag = flag.Bool("update", false, "Run the CLI in update mode")
	flag.Parse()

	if *installFlag || *updateFlag {
		if !<-GithubDoneChan {
			fmt.Println("Not", Ternary(*installFlag, "installing", "updating"), "as fetching release data failed")
			return
		}
	}

	if *installFlag {
		_ = PromptDiscord("patch").patch()
	} else if *uninstallFlag {
		_ = PromptDiscord("unpatch").unpatch()
	} else if *updateFlag {
		_ = installLatestBuilds()
	} else {
		flag.Usage()
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
