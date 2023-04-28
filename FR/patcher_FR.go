/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ProtonMail/go-appdir"
	"os"
	"os/exec"
	path "path/filepath"
	"strings"
)

var BaseDir string
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
		BaseDir = dir
	} else if dir = os.Getenv("DISCORD_USER_DATA_DIR"); dir != "" {
		fmt.Println("Using DISCORD_USER_DATA_DIR/../VencordData")
		BaseDir = path.Join(dir, "..", "VencordData")
	} else {
		fmt.Println("Using UserConfig")
		BaseDir = appdir.New("Vencord").UserConfig()
	}
	FilesDir = path.Join(BaseDir, "dist")
	if !ExistsFile(FilesDir) {
		FilesDirErr = os.MkdirAll(FilesDir, 0755)
		if FilesDirErr != nil {
			fmt.Println("Impossible de créér", FilesDir, FilesDirErr)
		} else {
			FilesDirErr = FixOwnership(BaseDir)
		}
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
	isOpenAsar       *bool
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
			return errors.New("Ce fichier: '" + name + "' n'appartient pas à Vencord")
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

func patchRenames(dir string, isSystemElectron bool) (err error) {
	appAsar := path.Join(dir, "app.asar")
	_appAsar := path.Join(dir, "_app.asar")

	var renamesDone [][]string
	defer func() {
		if err != nil && len(renamesDone) > 0 {
			fmt.Println("Erreur lors du patch, on annule les changements")
			for _, rename := range renamesDone {
				if innerErr := os.Rename(rename[1], rename[0]); innerErr != nil {
					fmt.Println("Erreur lors de l'annulation des changements. Merci de réinstaller", innerErr)
				} else {
					fmt.Println("Changements annulées.")
				}
			}
		}
	}()

	fmt.Println("On renomme", appAsar, "en", _appAsar)
	if err := os.Rename(appAsar, _appAsar); err != nil {
		err = CheckIfErrIsCauseItsBusyRn(err)
		fmt.Println(err)
		return err
	}
	renamesDone = append(renamesDone, []string{appAsar, _appAsar})

	if isSystemElectron {
		from, to := appAsar+".unpacked", _appAsar+".unpacked"
		fmt.Println("On renomme", from, "en", to)
		err := os.Rename(from, to)
		if err != nil {
			return err
		}
		renamesDone = append(renamesDone, []string{from, to})
	}

	fmt.Println("On écrit ici:", appAsar)
	if err := writeFiles(appAsar); err != nil {
		return err
	}

	return nil
}

func (di *DiscordInstall) patch() error {
	fmt.Println("On Patch " + di.path + "...")
	if LatestHash != InstalledHash {
		if err := InstallLatestBuilds(); err != nil {
			return nil // already shown dialog so don't return same error again
		}
	}

	if di.isPatched {
		fmt.Println(di.path, "est déjà patché, on le dépatch")
		if err := di.unpatch(); err != nil {
			if errors.Is(err, os.ErrPermission) {
				return err
			}
			return errors.New("erreur lors du dépatch dans ce dossier '" + di.path + "':\n" + err.Error())
		}
	}

	if di.isSystemElectron {
		if err := patchRenames(di.path, true); err != nil {
			return err
		}
	} else {
		for _, version := range di.versions {
			if err := patchRenames(path.Join(version, ".."), false); err != nil {
				return err
			}
		}
	}
	fmt.Println("Patché avec succès", di.path)
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

		fmt.Println("C'est un flatpak. On essaye de donner l'accès à Flatpak à", FilesDir+"...")

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
			fmt.Println("Ceci est une install personnelle, mais nous somme le root du système. Essayez de lancer la commande su pour lancer le programme en temps que", actualUser)
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
			return errors.New("Impossible de donner à DIscord Flatpak les permissions pour ce dossier: " + FilesDir + ": " + err.Error())
		}
	}
	return nil
}

func unpatchRenames(dir string, isSystemElectron bool) (errOut error) {
	appAsar := path.Join(dir, "app.asar")
	appAsarTmp := path.Join(dir, "app.asar.tmp")
	_appAsar := path.Join(dir, "_app.asar")

	var renamesDone [][]string
	defer func() {
		if errOut != nil && len(renamesDone) > 0 {
			fmt.Println("Erreur lors du dépatch, on annule les changements")
			for _, rename := range renamesDone {
				if innerErr := os.Rename(rename[1], rename[0]); innerErr != nil {
					fmt.Println("Erreur lors de l'annulation des changements. Merci de réinstaller", innerErr)
				} else {
					fmt.Println("Changements annulées.")
				}
			}
		} else if errOut == nil {
			if innerErr := os.RemoveAll(appAsarTmp); innerErr != nil {
				fmt.Println("Erreur lors de la supprésion de app.asar backup. Ce n'est pas obligé mais pouvez le supprimer manuellement", innerErr)
			}
		}
	}()

	fmt.Println("On supprime", appAsar)
	if err := os.Rename(appAsar, appAsarTmp); err != nil {
		err = CheckIfErrIsCauseItsBusyRn(err)
		fmt.Println(err)
		errOut = err
	} else {
		renamesDone = append(renamesDone, []string{appAsar, appAsarTmp})
	}

	fmt.Println("On renomme", _appAsar, "en", appAsar)
	if err := os.Rename(_appAsar, appAsar); err != nil {
		err = CheckIfErrIsCauseItsBusyRn(err)
		fmt.Println(err)
		errOut = err
	} else {
		renamesDone = append(renamesDone, []string{_appAsar, appAsar})
	}

	if isSystemElectron {
		fmt.Println("On renomme", _appAsar+".unpacked", "en", appAsar+".unpacked")
		if err := os.Rename(_appAsar+".unpacked", appAsar+".unpacked"); err != nil {
			fmt.Println(err)
			errOut = err
		}
	}
	return
}

func (di *DiscordInstall) unpatch() error {
	fmt.Println("On dépatch " + di.path + "...")

	if di.isSystemElectron {
		fmt.Println("Installation Electron système, on dépatche les renames")
		// See comment in Patch
		if err := unpatchRenames(di.path, true); err != nil {
			return err
		}
	} else {
		for _, version := range di.versions {
			isCanaryHack := IsDirectory(path.Join(version, "..", "app.asar"))
			if isCanaryHack {
				if err := unpatchRenames(path.Join(version, ".."), false); err != nil {
					return err
				}
			} else {
				err := IsSafeToDelete(version)
				if errors.Is(err, os.ErrPermission) {
					fmt.Println("Permission de lire", version, "refusée")
					return err
				}
				fmt.Println("On vérifie si ", version, "est safe à supprimer", Ternary(err == nil, "Oui", "Non"))
				if err != nil {
					return errors.New("Supprimer le dossier du patch version :  '" + version + "' est potentiellement dangereux, merci de le faire manuellement " + err.Error())
				}
				fmt.Println("On supprime", version)
				err = os.RemoveAll(version)
				if err != nil {
					return err
				}
			}
		}
	}
	fmt.Println("Le Patch a été supprimé avec succès!", di.path)
	di.isPatched = false
	return nil
}
