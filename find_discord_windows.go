/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"os"
	path "path/filepath"
	"strings"
	"sync"
)

var windowsNames = map[string]string{
	"stable": "Discord",
	"ptb":    "DiscordPTB",
	"canary": "DiscordCanary",
	"dev":    "DiscordDevelopment",
}

var killLock sync.Mutex

func ParseDiscordNew(p, branch string, isFlatpak bool) *DiscordInstall {
	return nil
}

func ParseDiscord(p, branch string) *DiscordInstall {
	entries, err := os.ReadDir(p)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			Log.Warn("Error during readdir "+p+":", err)
		}
		return nil
	}

	isPatched := false
	resourcesPath := ""
	for _, dir := range entries {
		if dir.IsDir() && strings.HasPrefix(dir.Name(), "app-") {
			resources := path.Join(p, dir.Name(), "resources")
			if !ExistsFile(resources) {
				continue
			}
			if resources > resourcesPath {
				resourcesPath = resources
				isPatched = isResourcesFolderPatched(resources)
			}
		}
	}

	if resourcesPath == "" {
		return nil
	}

	if branch == "" {
		branch = GetBranch(p)
	}

	return &DiscordInstall{
		path:          p,
		branch:        branch,
		resourcesPath: resourcesPath,
		isPatched:     isPatched,
		isFlatpak:     false,
	}
}

func FindDiscords() []any {
	var discords []any

	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		Log.Error("%LOCALAPPDATA% is empty???????")
		return discords
	}

	for branch, dirname := range windowsNames {
		p := path.Join(appData, dirname)
		if discord := ParseDiscord(p, branch); discord != nil {
			Log.Debug("Found Discord install at ", p)
			discords = append(discords, discord)
		}
	}
	return discords
}

func PreparePatch(di *DiscordInstall) {
	killLock.Lock()
	defer killLock.Unlock()

	info := di.findRunningInstance()
	if info == nil {
		Log.Debug("No running Discord process found for", di.path)
		return
	}

	if di.relaunchInfo == nil {
		di.relaunchInfo = info
	}

	Log.Debug("Stopping running Discord instance at", di.path, "pids:", info.pids)
	if err := killRunningInstance(di, info); err != nil {
		Log.Warn("Failed to stop Discord:", err)
	}
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
