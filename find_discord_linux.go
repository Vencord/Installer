/*
 * This part is file of VencordInstaller
 * Copyright (c) 2022 Vendicated
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"errors"
	"fmt"
	"os"
	path "path/filepath"
	"strings"
)

var (
	Home        string
	DiscordDirs []string
)

func init() {
	// If ran as sudo, the HOME environment variable will be that of root.
	// Thankfully, sudo sets the SUDO_USER env variable, so use that to look up
	// the actual HOME
	var sudoUser = os.Getenv("SUDO_USER")
	fmt.Println(sudoUser)
	if sudoUser != "" {
		passwd, err := ReadFile("/etc/passwd")
		if err != nil {
			// TODO
		}
		for _, line := range strings.Fields(passwd) {
			fmt.Println(line)
			if strings.HasPrefix(line, sudoUser+":") {
				fmt.Println("Found line")
				Home = strings.Split(line, ":")[5]
				// Error = invalid key but that won't ever happen
				_ = os.Setenv("HOME", Home)
				break
			}
		}
		// somehow not found?
		Home = os.Getenv("HOME")
	} else {
		Home = os.Getenv("HOME")
	}

	DiscordDirs = []string{
		"/usr/share",
		"/usr/lib64",
		"/opt",
		path.Join(Home, ".local/share"),
		"/var/lib/flatpak/app",
		path.Join(Home, "/.local/share/flatpak/app"),
	}
}

func ParseDiscord(p, _ string) *DiscordInstall {
	name := path.Base(p)

	isFlatpak := strings.Contains(p, "/flatpak/")
	if isFlatpak {
		discordName := name[len("com.discordapp."):]
		if discordName != "Discord" { //
			// DiscordCanary -> discord-canary
			discordName = strings.ToLower(discordName[:7] + "-" + discordName[7:])
		}
		p = path.Join(p, "current", "active", "files", discordName)
	}

	resources := path.Join(p, "resources")
	app := path.Join(resources, "app")

	isPatched, isSystemElectron := false, false

	if ExistsFile(resources) { // normal install
		isPatched = ExistsFile(app)
	} else if ExistsFile(path.Join(p, "app.asar")) { // System electron doesn't have resources folder
		isSystemElectron = true
		isPatched = ExistsFile(path.Join(p, "_app.asar.unpacked"))
	} else {
		return nil
	}

	return &DiscordInstall{
		path:             p,
		branch:           GetBranch(name),
		versions:         []string{app},
		isPatched:        isPatched,
		isFlatpak:        isFlatpak,
		isSystemElectron: isSystemElectron,
	}
}

func FindDiscords() []any {
	var discords []any
	for _, dir := range DiscordDirs {
		children, err := os.ReadDir(dir)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				fmt.Println("Error during readdir "+dir+":", err)
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
				discords = append(discords, discord)
			}
		}
	}

	return discords
}
