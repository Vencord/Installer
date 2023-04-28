/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"fmt"
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
		versions:         []string{app},
		isPatched:        ExistsFile(app) || IsDirectory(path.Join(resources, "app.asar")),
		isFlatpak:        false,
		isSystemElectron: false,
	}
}

func FindDiscords() []any {
	var discords []any
	for branch, dirname := range macosNames {
		p := "/Applications/" + dirname
		if discord := ParseDiscord(p, branch); discord != nil {
			fmt.Println("Found Discord Install at", p)
			discords = append(discords, discord)
		}
	}
	return discords
}

func FixOwnership(_ string) error {
	return nil
}

func CheckScuffedInstall() bool {
	return false
}
