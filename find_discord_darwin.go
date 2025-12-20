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

func ParseDiscord(p, branch string) *DiscordInstall {
	if !ExistsFile(p) {
		return nil
	}

	resources := path.Join(p, "/Contents/Resources")
	if !ExistsFile(resources) {
		return nil
	}

	if branch == "" {
		branch = GetBranch(strings.TrimSuffix(p, ".app"))
	}

	app := path.Join(resources, "app")
	return &DiscordInstall{
		path:             p,
		branch:           branch,
		appPath:          app,
		isPatched:        ExistsFile(path.Join(resources, "_app.asar")),
		isFlatpak:        false,
		isSystemElectron: false,
	}
}

func FindDiscords() []any {
	var discords []any
	bases := []string{
		"/Applications",
		path.Join(os.Getenv("HOME"), "Applications"),
	}
	
	handleDiscordDir := func(p string, branch string) {
		if discord := ParseDiscord(p, branch); discord != nil {
			Log.Debug("Found Discord install at ", p)
			discords = append(discords, discord)
		}
	}

	for branch, dirname := range macosNames {
		for _, base := range bases {
			p := path.Join(base, dirname)
			handleDiscordDir(p, branch)
		}
	}

	discordCustomDir := os.Getenv("VENCORD_DISCORD_DIR")
	if discordCustomDir != "" {
		handleDiscordDir(discordCustomDir, "custom")
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
