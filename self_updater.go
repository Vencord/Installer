/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"fmt"
	"github.com/Vendicated/VencordInstaller/buildinfo"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"
)

var IsSelfOutdated = false

func init() {
	go DeleteOldExecutable()
}

func CheckSelfUpdate() {
	Log.Debug("Checking for Installer Updates...")

	res, err := GetGithubRelease(InstallerReleaseUrl, InstallerReleaseUrlFallback)
	if err == nil {
		IsSelfOutdated = res.TagName != buildinfo.InstallerTag
	}
}

func GetInstallerDownloadLink() string {
	const BaseUrl = "https://github.com/Vencord/Installer/releases/latest/download/"
	switch runtime.GOOS {
	case "windows":
		filename := Ternary(buildinfo.UiType == buildinfo.UiTypeCli, "VencordInstallerCli.exe ", "VencordInstaller.exe ")
		return BaseUrl + filename
	case "darwin":
		return BaseUrl + "VencordInstaller.MacOS.zip"
	case "linux":
		return BaseUrl + " VencordInstallerCli-linux "
	default:
		return ""
	}
}

func GetInstallerDownloadMarkdown() string {
	link := GetInstallerDownloadLink()
	if link == "" {
		return ""
	}
	return " [Download the latest Installer](" + link + ")"
}

func CanUpdateSelf() bool {
	//goland:noinspection GoBoolExpressions
	return IsSelfOutdated && runtime.GOOS != "darwin"
}

func UpdateSelf() error {
	if !CanUpdateSelf() {
		return errors.New("Cannot update self. Either no update available or macos")
	}

	url := GetInstallerDownloadLink()
	if url == "" {
		return errors.New("Failed to get installer download link")
	}

	Log.Debug("Updating self from", url)

	ownExePath, err := os.Executable()
	if err != nil {
		return err
	}

	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	tmp, err := os.CreateTemp("", "VencordInstallerUpdate")
	if err != nil {
		return fmt.Errorf("Failed to create tempfile: %w", err)
	}

	if _, err = io.Copy(tmp, res.Body); err != nil {
		return err
	}

	if err = os.Remove(ownExePath); err != nil {
		if err = os.Rename(ownExePath, ownExePath+".old"); err != nil {
			return fmt.Errorf("Failed to remove/rename own executable: %w", err)
		}
	}

	if err = os.Rename(tmp.Name(), ownExePath); err != nil {
		return fmt.Errorf("Failed to replace self with updated executable. Please manually redownload the installer: %w", err)
	}

	return nil
}

func DeleteOldExecutable() {
	ownExePath, err := os.Executable()
	if err != nil {
		return
	}

	for attempts := 0; attempts < 10; attempts += 1 {
		err = os.Remove(ownExePath + ".old")

		if err == nil || errors.Is(err, os.ErrNotExist) {
			break
		}

		Log.Warn("Failed to remove old executable. Retrying in 1 second.", err)
		time.Sleep(1 * time.Second)
	}
}
