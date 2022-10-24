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
	"fmt"
	"os"
	"path"

	"github.com/ProtonMail/go-appdir"
)

var FilesDir string
var FilesDirErr error

func init() {
	if dir := os.Getenv("VENCORD_USER_DATA_DIR"); dir != "" {
		FilesDir = path.Join(dir, "dist")
	} else if dir = os.Getenv("DISCORD_USER_DATA_DIR"); dir != "" {
		FilesDir = path.Join(dir, "..", "VencordData", "dist")
	} else {
		FilesDir = path.Join(appdir.New("Vencord").UserConfig(), "dist")
	}
	FilesDirErr = os.MkdirAll(FilesDir, 0755)
	if FilesDirErr != nil {
		fmt.Println("Failed to create", FilesDir, FilesDirErr)
	}
}

type DiscordInstall struct {
	path      string   // the full path
	branch    string   // the branch like canary/stable
	versions  []string // windows may have multiple versions, just patch all to make sure
	isPatched bool
	isFlatpak bool
}

func patch(path string) error {
	return nil
}

func unpatch(path string) error {
	return nil
}
