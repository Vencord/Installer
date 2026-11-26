/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"os"
	"os/exec"
	path "path/filepath"
	"regexp"
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

var nextAppVersion = regexp.MustCompile(`(?m)^[ \t]*next[ \t]*=[ \t]*"([0-9]+\.[0-9]+\.[0-9]+)"[ \t]*(?:#[^\n]*)?\r?$`)

// Supposedly, the discord folder in Application Support may disallow enumeration, but full file reads are allowed
// We still need to find out the app version to be able to access it.
// There are two files that tell us this info:
// discord/install-state/app.toml ~ A simple toml file with a next value
// discord/install.db ~ An sqlite database containing json in two rows. Why is this a sqlite database instead of just a json file? Who knows.
// Either way, we try to read them to get the version. Thankfully mac comes with sqlite3 preinstalled so we can just shell out

func findMacSupportInstalls(dir, bundle, branch string) []any {
	var installs []any
	addBundle := func(appDir string) bool {
		p := path.Join(dir, appDir, bundle)
		install := ParseDiscord(p, branch)
		if install == nil {
			return false
		}
		Log.Debug("Found Discord Install at", p)
		installs = append(installs, install)
		return true
	}

	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "app-") {
				addBundle(entry.Name())
			}
		}
		return installs
	}
	if !os.IsNotExist(err) {
		Log.Warn("Failed to list Discord installs:", err)
	}
	if !os.IsPermission(err) {
		return nil
	}

	state, err := os.ReadFile(path.Join(dir, "install-state", "app.toml"))
	if err != nil {
		Log.Debug("Failed to read Discord install state:", err)
	} else if match := nextAppVersion.FindSubmatch(state); match != nil {
		if addBundle("app-" + string(match[1])) {
			return installs
		}
	}

	for _, version := range installedHostVersions(dir) {
		addBundle("app-" + version)
	}
	return installs
}

func installedHostVersions(dir string) []string {
	const query = `
		SELECT DISTINCT printf('%d.%d.%d',
			json_extract(host.value, '$.host_version.version[0]'),
			json_extract(host.value, '$.host_version.version[1]'),
			json_extract(host.value, '$.host_version.version[2]')
		)
		FROM key_values, json_each(key_values.value) AS host
		WHERE key_values.key GLOB 'host/app/*'
		  AND json_extract(host.value, '$.install_state') = 'Installed'
		  AND json_array_length(host.value, '$.host_version.version') = 3
	`
	db := path.Join(dir, "installer.db")
	cmd := exec.Command("/usr/bin/sqlite3",
		"-batch", "-init", "/dev/null", "-readonly", db, query,
	)
	versions, err := cmd.Output()
	if err != nil {
		Log.Debug("Failed to query", db, err)
		return nil
	}
	return strings.Fields(string(versions))
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

			dir := path.Join(supportBase, branchDir)
			discords = append(discords, findMacSupportInstalls(dir, dirname, branch)...)
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
