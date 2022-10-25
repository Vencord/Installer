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
	"errors"
	"fmt"
	"os"
	"os/exec"
	path "path/filepath"
	"strings"

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
		fmt.Println("Using VENCORD_USER_DATA_DIR")
		FilesDir = path.Join(dir, "dist")
	} else if dir = os.Getenv("DISCORD_USER_DATA_DIR"); dir != "" {
		fmt.Println("Using DISCORD_USER_DATA_DIR/../VencordData")
		FilesDir = path.Join(dir, "..", "VencordData", "dist")
	} else {
		fmt.Println("Using UserConfig")
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

// IsSafeToDelete returns nil if path is safe to delete.
// In other cases, the returned error should give more info
func IsSafeToDelete(path string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		return err
	}
	for _, file := range files {
		name := file.Name()
		if name != "package.json" && name != "index.js" {
			return errors.New("Found file '" + name + "' which doesn't belong to Vencord.")
		}
	}
	return nil
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

func handleErr(err error, action string, retry func(g.DialogResult)) {
	if errors.Is(err, os.ErrPermission) {
		err = errors.New("Permission denied. Maybe try running me as Administrator/Root?")
	}
	g.Msgbox("Uh Oh!", "Failed to "+action+" this Install:\n"+err.Error()+"\n\nRetry?").
		Buttons(g.MsgboxButtonsYesNo).
		ResultCallback(retry)
}

func (di *DiscordInstall) Patch() {
	if err := di.patch(); err != nil {
		handleErr(err, "patch", func(retry g.DialogResult) {
			if retry {
				di.Patch()
			}
		})
	}
}

func (di *DiscordInstall) Unpatch() {
	if err := di.unpatch(); err != nil {
		handleErr(err, "unpatch", func(retry g.DialogResult) {
			if retry {
				di.Unpatch()
			}
		})
	}
}

func (di *DiscordInstall) patch() error {
	fmt.Println("Patching " + di.path + "...")
	if LatestHash != InstalledHash {
		if err := InstallLatestBuilds(); err != nil {
			return nil // already shown dialog so don't return same error again
		}
	}

	if di.isPatched {
		fmt.Println(di.path, "is already patched. Unpatching first...")
		if err := di.unpatch(); err != nil {
			if errors.Is(err, os.ErrPermission) {
				return err
			}
			return errors.New("patch: Failed to unpatch already patched install '" + di.path + "':\n" + err.Error())
		}
	}

	if di.isSystemElectron {
		fmt.Println("Detected as System Electron Install")
		// hack:
		// - Rename app.asar 		  -> _app.asar
		// - Rename app.asar.unpacked -> _app.asar.unpacked
		// - Make app.asar folder with patch files
		// This breaks pacman but shrug. Someone with this setup should be smart enough to fix it
		// Perhaps I could register a pacman hook to fix it
		appAsar := path.Join(di.path, "app.asar")
		_appAsar := path.Join(di.path, "_app.asar")
		fmt.Println("Renaming", appAsar, "to", _appAsar)
		if err := os.Rename(appAsar, _appAsar); err != nil {
			return err
		}
		fmt.Println("Renaming", appAsar+".unpacked", "to", _appAsar+".unpacked")
		err := os.Rename(appAsar+".unpacked", _appAsar+".unpacked")
		if err != nil {
			return err
		}
		fmt.Println("Writing files to", appAsar)
		if err = writeFiles(appAsar); err != nil {
			return err
		}
	} else {
		for _, version := range di.versions {
			fmt.Println("Writing files to", version)
			if err := writeFiles(version); err != nil {
				return err
			}
		}
	}
	fmt.Println("Successfully patched", di.path)
	di.isPatched = true

	if di.isFlatpak {
		pathElements := strings.Split(di.path, "/")
		var name string
		for _, e := range pathElements {
			if strings.HasPrefix(e, "com.discordapp") {
				name = e
				break
			}
		}

		fmt.Println("This is a flatpak. Trying to grant the Flatpak access to", FilesDir+"...")

		isSystemFlatpak := strings.HasPrefix(di.path, "/var")
		var args []string
		if !isSystemFlatpak {
			args = append(args, "--user")
		}
		args = append(args, "override", name, "--filesystem="+FilesDir)
		fullCmd := "flatpak " + strings.Join(args, " ")

		fmt.Println("Running", fullCmd)

		var err error
		if !isSystemFlatpak && os.Getuid() == 0 {
			// We are operating on a user flatpak but are root
			actualUser := os.Getenv("SUDO_USER")
			fmt.Println("This is a user install but we are root. Using su to run as", actualUser)
			cmd := exec.Command("su", "-", actualUser, "-c", "sh", "-c", fullCmd)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
		} else {
			cmd := exec.Command("flatpak", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
		}
		if err != nil {
			return errors.New("Failed to grant Discord Flatpak access to " + FilesDir + ": " + err.Error())
		}
	}
	return nil
}

func (di *DiscordInstall) unpatch() error {
	fmt.Println("Unpatching " + di.path + "...")

	if di.isSystemElectron {
		fmt.Println("Detected as System Electron Install")
		// See comment in Patch
		appAsar := path.Join(di.path, "app.asar")
		_appAsar := path.Join(di.path, "_app.asar")
		fmt.Println("Deleting", appAsar)
		if err := os.RemoveAll(appAsar); err != nil {
			return err
		}
		fmt.Println("Renaming", _appAsar, "to", appAsar)
		if err := os.Rename(_appAsar, appAsar); err != nil {
			return err
		}
		fmt.Println("Renaming", _appAsar+".unpacked", "to", appAsar+".unpacked")
		if err := os.Rename(_appAsar+".unpacked", appAsar+".unpacked"); err != nil {
			return err
		}
	} else {
		for _, version := range di.versions {
			err := IsSafeToDelete(version)
			if errors.Is(err, os.ErrPermission) {
				fmt.Println("Permission to read", version, "denied")
				return err
			}
			fmt.Println("Checking if", version, "is safe to delete:", Ternary(err == nil, "Yes", "No"))
			if err != nil {
				return errors.New("Deleting patch folder '" + version + "' is possibly unsafe. Please do it manually: " + err.Error())
			}
			fmt.Println("Deleting", version)
			err = os.RemoveAll(version)
			if err != nil {
				return err
			}
		}
	}
	fmt.Println("Successfully unpatched", di.path)
	di.isPatched = false
	return nil
}
