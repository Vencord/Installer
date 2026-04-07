/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"github.com/ProtonMail/go-appdir"
	"os"
	"os/exec"
	path "path/filepath"
	"strings"
)

var BaseDir string
var VencordDirectory string

func init() {
	if dir := os.Getenv("VENCORD_USER_DATA_DIR"); dir != "" {
		Log.Debug("Using VENCORD_USER_DATA_DIR")
		BaseDir = dir
	} else if dir = os.Getenv("DISCORD_USER_DATA_DIR"); dir != "" {
		Log.Debug("Using DISCORD_USER_DATA_DIR/../VencordData")
		BaseDir = path.Join(dir, "..", "VencordData")
	} else {
		Log.Debug("Using UserConfig")
		BaseDir = appdir.New("Vencord").UserConfig()
	}

	if dir := os.Getenv("VENCORD_DIRECTORY"); dir != "" {
		Log.Debug("Using VENCORD_DIRECTORY")
		VencordDirectory = dir
	} else {
		VencordDirectory = path.Join(BaseDir, "vencord.asar")
	}
}

type DiscordInstall struct {
	path             string // the base path
	branch           string // canary / stable / ...
	appPath          string // List of app folder to patch
	isPatched        bool
	isFlatpak        bool
	isSystemElectron bool // Needs special care https://aur.archlinux.org/packages/discord_arch_electron
	isOpenAsar       *bool
}

//region Patch

func patchAppAsar(dir string, isSystemElectron bool) (err error) {
	appAsar := path.Join(dir, "app.asar")
	_appAsar := path.Join(dir, "_app.asar")

	var renamesDone [][]string
	defer func() {
		if err != nil && len(renamesDone) > 0 {
			Log.Error("Failed to patch. Undoing partial patch")
			for _, rename := range renamesDone {
				if innerErr := os.Rename(rename[1], rename[0]); innerErr != nil {
					Log.Error("Failed to undo partial patch. This install is probably bricked.", innerErr)
				} else {
					Log.Info("Successfully undid all changes")
				}
			}
		}
	}()

	Log.Debug("Renaming", appAsar, "to", _appAsar)
	if err := os.Rename(appAsar, _appAsar); err != nil {
		err = CheckIfErrIsCauseItsBusyRn(err)
		Log.Error(err.Error())
		return err
	}
	renamesDone = append(renamesDone, []string{appAsar, _appAsar})

	if isSystemElectron {
		from, to := appAsar+".unpacked", _appAsar+".unpacked"
		Log.Debug("Renaming", from, "to", to)
		err := os.Rename(from, to)
		if err != nil {
			return err
		}
		renamesDone = append(renamesDone, []string{from, to})
	}

	Log.Debug("Writing custom app.asar to", appAsar)
	if err := WriteAppAsar(appAsar, VencordDirectory); err != nil {
		return err
	}

	return nil
}

func (di *DiscordInstall) patch() error {
	Log.Info("Patching " + di.path + "...")
	if LatestHash != InstalledHash {
		if err := InstallLatestBuilds(); err != nil {
			return nil // already shown dialog so don't return same error again
		}
	}

	PreparePatch(di)

	if di.isPatched {
		Log.Info(di.path, "is already patched. Unpatching first...")
		if err := di.unpatch(); err != nil {
			if errors.Is(err, os.ErrPermission) {
				return err
			}
			return errors.New("patch: Failed to unpatch already patched install '" + di.path + "':\n" + err.Error())
		}
	}

	if di.isSystemElectron {
		if err := patchAppAsar(di.path, true); err != nil {
			return err
		}
	} else {
		if err := patchAppAsar(path.Join(di.appPath, ".."), false); err != nil {
			return err
		}
	}

	Log.Info("Successfully patched", di.path)
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

		Log.Debug("This is a flatpak. Trying to grant the Flatpak access to", VencordDirectory+"...")

		isSystemFlatpak := strings.HasPrefix(di.path, "/var")
		var args []string
		if !isSystemFlatpak {
			args = append(args, "--user")
		}
		args = append(args, "override", name, "--filesystem="+VencordDirectory)
		fullCmd := "flatpak " + strings.Join(args, " ")

		Log.Debug("Running", fullCmd)

		var err error
		if !isSystemFlatpak && os.Getuid() == 0 {
			// We are operating on a user flatpak but are root
			actualUser := os.Getenv("SUDO_USER")
			Log.Debug("This is a user install but we are root. Using su to run as", actualUser)
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
			return errors.New("Failed to grant Discord Flatpak access to " + VencordDirectory + ": " + err.Error())
		}
	}
	return nil
}

//endregion

// region Unpatch

func unpatchAppAsar(dir string, isSystemElectron bool) (errOut error) {
	appAsar := path.Join(dir, "app.asar")
	appAsarTmp := path.Join(dir, "app.asar.tmp")
	_appAsar := path.Join(dir, "_app.asar")

	var renamesDone [][]string
	defer func() {
		if errOut != nil && len(renamesDone) > 0 {
			Log.Error("Failed to unpatch. Undoing partial unpatch")
			for _, rename := range renamesDone {
				if innerErr := os.Rename(rename[1], rename[0]); innerErr != nil {
					Log.Error("Failed to undo partial unpatch. This install is probably bricked.", innerErr)
				} else {
					Log.Info("Successfully undid all changes")
				}
			}
		} else if errOut == nil {
			if innerErr := os.RemoveAll(appAsarTmp); innerErr != nil {
				Log.Warn("Failed to delete temporary app.asar (patch folder) backup. This is whatever but you might want to delete it manually.", innerErr)
			}
		}
	}()

	Log.Debug("Deleting", appAsar)
	if err := os.Rename(appAsar, appAsarTmp); err != nil {
		err = CheckIfErrIsCauseItsBusyRn(err)
		Log.Error(err.Error())
		errOut = err
	} else {
		renamesDone = append(renamesDone, []string{appAsar, appAsarTmp})
	}

	Log.Debug("Renaming", _appAsar, "to", appAsar)
	if err := os.Rename(_appAsar, appAsar); err != nil {
		err = CheckIfErrIsCauseItsBusyRn(err)
		Log.Error(err.Error())
		errOut = err
	} else {
		renamesDone = append(renamesDone, []string{_appAsar, appAsar})
	}

	if isSystemElectron {
		Log.Debug("Renaming", _appAsar+".unpacked", "to", appAsar+".unpacked")
		if err := os.Rename(_appAsar+".unpacked", appAsar+".unpacked"); err != nil {
			Log.Error(err.Error())
			errOut = err
		}
	}
	return
}

func (di *DiscordInstall) unpatch() error {
	Log.Info("Unpatching " + di.path + "...")

	PreparePatch(di)

	if di.isSystemElectron {
		if err := unpatchAppAsar(di.path, true); err != nil {
			return err
		}
	} else {
		if err := unpatchAppAsar(path.Join(di.appPath, ".."), false); err != nil {
			return err
		}
	}

	Log.Info("Successfully unpatched", di.path)
	di.isPatched = false
	return nil
}

//endregion
