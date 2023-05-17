/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	path "path/filepath"
	"strings"
)

var windowsNames = map[string]string{
	"stable": "Discord",
	"ptb":    "DiscordPTB",
	"canary": "DiscordCanary",
	"dev":    "DiscordDevelopment",
}

func ParseDiscord(p, branch string) *DiscordInstall {
	entries, err := os.ReadDir(p)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Println("Error during readdir "+p+":", err)
		}
		return nil
	}

	isPatched := false
	appPath := ""
	for _, dir := range entries {
		if dir.IsDir() && strings.HasPrefix(dir.Name(), "app-") {
			resources := path.Join(p, dir.Name(), "resources")
			if !ExistsFile(resources) {
				continue
			}
			app := path.Join(resources, "app")
			if app > appPath {
				appPath = app
				isPatched = ExistsFile(app) || IsDirectory(path.Join(resources, "app.asar"))
			}
		}
	}

	if appPath == "" {
		return nil
	}

	if branch == "" {
		branch = GetBranch(p)
	}

	return &DiscordInstall{
		path:             p,
		branch:           branch,
		appPath:          appPath,
		isPatched:        isPatched,
		isFlatpak:        false,
		isSystemElectron: false,
	}
}

func FindDiscords() []any {
	var discords []any

	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		fmt.Println("%LOCALAPPDATA% is empty???????")
		return discords
	}

	for branch, dirname := range windowsNames {
		p := path.Join(appData, dirname)
		if discord := ParseDiscord(p, branch); discord != nil {
			fmt.Println("Found Discord install at ", p)
			discords = append(discords, discord)
		}
	}
	return discords
}

func PreparePatch(di *DiscordInstall) {
	name := windowsNames[di.branch]
	fmt.Println("Killing " + name + "...")

	_ = exec.Command("powershell", "Stop-Process -Name "+name).Run()
}

func FixOwnership(_ string) error {
	return nil
}

// https://github.com/Vencord/Installer/issues/9

func CheckScuffedInstall() bool {
	username := os.Getenv("USERNAME")
	programData := os.Getenv("PROGRAMDATA")
	for _, discordName := range windowsNames {
		if ExistsFile(path.Join(programData, username, discordName)) || ExistsFile(path.Join(programData, username, discordName)) {
			HandleScuffedInstall()
			return true
		}
	}
	return false
}
