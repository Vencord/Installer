/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"io/fs"
	"os"
	"os/user"
	path "path/filepath"
	"strconv"
	"strings"
)

var (
	Home        string
	DiscordDirs []string
)

func init() {
	// If ran as root, the HOME environment variable will be that of root.
	// SUDO_USER and DOAS_USER tell us the actual user
	var sudoUser = os.Getenv("SUDO_USER")
	if sudoUser == "" {
		sudoUser = os.Getenv("DOAS_USER")
		if sudoUser != "" {
			_ = os.Setenv("SUDO_USER", sudoUser)
		}
	}
	if sudoUser != "" {
		if sudoUser == "root" {
			panic("VencordInstaller must not be run as the root user. Please rerun as normal user. Use sudo or doas to run as root.")
		}

		Log.Debug("VencordInstaller was run with root privileges, actual user is", sudoUser)
		Log.Debug("Looking up HOME of", sudoUser)

		u, err := user.Lookup(sudoUser)
		if err != nil {
			Log.Warn("Failed to lookup HOME", err)
		} else {
			Log.Debug("Actual HOME is", u.HomeDir)
			_ = os.Setenv("HOME", u.HomeDir)
		}
	} else if os.Getuid() == 0 {
		panic("VencordInstaller was run as root but neither SUDO_USER nor DOAS_USER are set. Please rerun me as a normal user, with sudo/doas, or manually set SUDO_USER to your username")
	}
	Home = os.Getenv("HOME")

	DiscordDirs = []string{
		"/usr/share",
		"/usr/lib64",
		"/opt",
		path.Join(Home, ".local/share"),
		path.Join(Home, ".dvm"),
		"/var/lib/flatpak/app",
		path.Join(Home, "/.local/share/flatpak/app"),
	}
}

func ParseDiscord(p, _ string) *DiscordInstall {
	name := path.Base(p)

	needsFlatpakResolve := strings.Contains(p, "/flatpak/") && !strings.Contains(p, "/current/active/files/")
	if needsFlatpakResolve {
		discordName := strings.ToLower(name[len("com.discordapp."):])
		if discordName != "discord" { //
			// DiscordCanary -> discord-canary
			discordName = discordName[:7] + "-" + discordName[7:]
		}
		p = path.Join(p, "current/active/files", discordName)
	}

	resources := path.Join(p, "resources")
	app := path.Join(resources, "app")

	isPatched, isSystemElectron := false, false

	if ExistsFile(resources) { // normal install
		isPatched = ExistsFile(app) || IsDirectory(path.Join(resources, "app.asar"))
	} else if ExistsFile(path.Join(p, "app.asar")) { // System electron doesn't have resources folder
		isSystemElectron = true
		isPatched = ExistsFile(path.Join(p, "_app.asar.unpacked"))
	} else {
		Log.Warn("Tried to parse invalid Location:", p)
		return nil
	}

	return &DiscordInstall{
		path:             p,
		branch:           GetBranch(name),
		appPath:          app,
		isPatched:        isPatched,
		isFlatpak:        needsFlatpakResolve,
		isSystemElectron: isSystemElectron,
	}
}

func FindDiscords() []any {
	var discords []any
	for _, dir := range DiscordDirs {
		children, err := os.ReadDir(dir)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				Log.Warn("Error during readdir "+dir+":", err)
			}
			continue
		}

		for _, child := range children {
			name := child.Name()
			if !child.IsDir() || !ArrayIncludes(LinuxDiscordNames, name) {
				continue
			}

			discordDir := path.Join(dir, name)
			if discord := ParseDiscord(discordDir, ""); discord != nil {
				Log.Debug("Found Discord install at ", discordDir)
				discords = append(discords, discord)
			}
		}
	}

	return discords
}

func PreparePatch(di *DiscordInstall) {}

// FixOwnership fixes file ownership on Linux
func FixOwnership(p string) error {
	if os.Geteuid() != 0 {
		return nil
	}

	Log.Debug("Fixing Ownership of", p)

	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" {
		panic("SUDO_USER was empty. This point should never be reached")
	}

	Log.Debug("Looking up User", sudoUser)
	u, err := user.Lookup(sudoUser)
	if err != nil {
		Log.Error("Lookup failed:", err)
		return err
	}
	Log.Debug("Lookup successful, Uid", u.Uid, "Gid", u.Gid)
	// This conversion is safe because of the GOOS guard above
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	err = path.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
		if err == nil {
			err = os.Chown(path, uid, gid)
			Log.Debug("chown", u.Uid+":"+u.Gid, path+":", Ternary(err == nil, "Success!", "Failed"))
		}
		return err
	})

	if err != nil {
		Log.Error("Failed to fix ownership:", err)
	}
	return err
}

func CheckScuffedInstall() bool {
	return false
}
