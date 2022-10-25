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
	var versions []string
	for _, dir := range entries {
		if dir.IsDir() && strings.HasPrefix(dir.Name(), "app-") {
			resources := path.Join(p, dir.Name(), "resources")
			if !ExistsFile(resources) {
				continue
			}
			app := path.Join(p, "app")
			versions = append(versions, app)
			isPatched = isPatched || ExistsFile(app)
		}
	}

	if len(versions) == 0 {
		return nil
	}

	if branch == "" {
		branch = GetBranch(p)
	}
	
	return &DiscordInstall{
		path:             p,
		branch:           branch,
		versions:         versions,
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
			discords = append(discords, discord)
		}
	}
	return discords
}
