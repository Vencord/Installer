/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"os"
	path "path/filepath"
	"strings"
)

var macosNames = map[string]string{
	"stable": "Discord.app",
	"ptb":    "Discord PTB.app",
	"canary": "Discord Canary.app",
	"dev":    "Discord Development.app",
}

func ParseDiscordNew(p, branch string, isFlatpak bool) *DiscordInstall {
	return nil
}

func ParseDiscord(p, branch string) *DiscordInstall {
	if !ExistsFile(p) {
		return nil
	}

	resources := path.Join(p, "/Contents/Resources")
	if !isResourcesFolderValid(resources) {
		return nil
	}

	if branch == "" {
		branch = GetBranch(strings.TrimSuffix(p, ".app"))
	}

	return &DiscordInstall{
		path:          p,
		branch:        branch,
		resourcesPath: resources,
		isPatched:     isResourcesFolderPatched(resources),
		isFlatpak:     false,
	}
}

func FindDiscords() []any {
	var discords []any
	bases := []string{
		"/Applications",
		path.Join(os.Getenv("HOME"), "Applications"),
	}
	supportBase := path.Join(os.Getenv("HOME"), "Library", "Application Support")

	for branch, dirname := range macosNames {
		foundInApps := false

		for _, base := range bases {
			p := path.Join(base, dirname)
			if discord := ParseDiscord(p, branch); discord != nil {
				Log.Debug("Found Discord Install at", p)
				discords = append(discords, discord)
				foundInApps = true
			}
		}

		if !foundInApps {
			branchDir := "discord" + branch
			if branch == "stable" {
				branchDir = "discord"
			}

			matches, err := path.Glob(path.Join(supportBase, branchDir, "app-*", dirname))
			if err == nil {
				for _, match := range matches {
					if discord := ParseDiscord(match, branch); discord != nil {
						Log.Debug("Found Discord Install at", match)
						discords = append(discords, discord)
					}
				}
			}
		}
	}
	return discords
}

func PreparePatch(di *DiscordInstall) {}

func FixOwnership(_ string) error {
	return nil
}

func CheckScuffedInstall() bool {
	return false
}
