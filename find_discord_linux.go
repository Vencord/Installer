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
	"path"
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
	if sudoUser != "" {
		passwd, err := ReadFile("/etc/passwd")
		if err != nil {
			// TODO
		}
		for _, line := range strings.Fields(passwd) {
			if strings.HasPrefix(line, sudoUser+":") {
				Home = strings.Split(line, ":")[5]
				break
			}
			// somehow not found?
			Home = os.Getenv("HOME")
		}
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

// FindDiscords actually returns []string but gui wants any
func FindDiscords() []any {
	var discords []any
	for _, discordDir := range DiscordDirs {
		entries, err := os.ReadDir(discordDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else {
				// handle maybe?
			}
		}
		for _, entry := range entries {
			name := entry.Name()
			if !entry.IsDir() || !ArrayIncludes(BranchNames, name) {
				continue
			}
			discords = append(discords, path.Join(discordDir, name))
		}
	}
	fmt.Println(discords)
	return discords
}
