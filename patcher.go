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
	"encoding/json"
	"fmt"
	"os"
	"path"

	g "github.com/AllenDang/giu"
	"github.com/ProtonMail/go-appdir"
)

var FilesDir string
var FilesDirErr error
var Patcher string

var PackageJson = []byte(`{
	"name": "discord",
	"main": "index.js"
}
`)

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
	Patcher = path.Join(FilesDir, "patcher.js")
}

type DiscordInstall struct {
	path             string   // the base path
	branch           string   // canary / stable / ...
	versions         []string // List of paths to folders to patch, 1 on linux/mac, might be more on Windows
	isPatched        bool
	isFlatpak        bool
	isSystemElectron bool // Needs special care https://aur.archlinux.org/packages/discord_arch_electron
}

func writeFiles(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}

	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(dir, "package.json"), PackageJson, 0644); err != nil {
		return err
	}

	patcherPath, _ := json.Marshal(Patcher)
	return os.WriteFile(path.Join(dir, "index.js"), []byte("require("+string(patcherPath)+")"), 0644)
}

func (di *DiscordInstall) Patch() {
	if err := di.patch(); err != nil {
		g.Msgbox("Uh Oh!", "Failed to patch this Install:\n"+err.Error()+"\n\nRetry?").
			Buttons(g.MsgboxButtonsYesNo).
			ResultCallback(func(retry g.DialogResult) {
				if retry {
					di.Patch()
				}
			})
	}
}

func (di *DiscordInstall) Unpatch() {
	if err := di.patch(); err != nil {
		g.Msgbox("Uh Oh!", "Failed to unpatch this Install:\n"+err.Error()+"\n\nRetry?").
			Buttons(g.MsgboxButtonsYesNo).
			ResultCallback(func(retry g.DialogResult) {
				if retry {
					di.Unpatch()
				}
			})
	}
}

func (di *DiscordInstall) patch() error {
	if err := InstallLatestBuilds(); err != nil {
		return nil // already shown dialog so don't return same error again
	}

	if di.isSystemElectron {
		// hack:
		// - Rename app.asar 		  -> _app.asar
		// - Rename app.asar.unpacked -> _app.asar.unpacked
		// - Make app.asar folder with patch files
		// This breaks pacman but shrug. Someone with this setup should be smart enough to fix it
		// Perhaps I could register a pacman hook to fix it
		appAsar := path.Join(di.path, "app.asar")
		_appAsar := path.Join(di.path, "_app.asar")
		if err := os.Rename(appAsar, _appAsar); err != nil {
			return err
		}
		err := os.Rename(appAsar+".unpacked", _appAsar+".unpacked")
		if err != nil {
			return err
		}
		if err = writeFiles(appAsar); err != nil {
			return err
		}
	} else {
		for _, version := range di.versions {
			if err := writeFiles(version); err != nil {
				return err
			}
		}
	}
	di.isPatched = true
	return nil
}

func (di *DiscordInstall) unpatch() error {
	if di.isSystemElectron {
		// See comment in Patch
		appAsar := path.Join(di.path, "app.asar")
		_appAsar := path.Join(di.path, "_app.asar")
		if err := os.RemoveAll(appAsar); err != nil {
			return err
		}
		if err := os.Rename(_appAsar, appAsar); err != nil {
			return err
		}
		if err := os.Rename(_appAsar+".unpacked", appAsar+".unpacked"); err != nil {
			return err
		}
	} else {
		for _, version := range di.versions {
			err := os.RemoveAll(version)
			if err != nil {
				return err
			}
		}
	}
	di.isPatched = false
	return nil
}
