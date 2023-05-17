/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"fmt"
	"runtime"
)

var IsInstallerOutdated = false

func CheckSelfUpdate() {
	fmt.Println("Checking for Installer Updates...")

	res, err := GetGithubRelease(InstallerReleaseUrl, InstallerReleaseUrlFallback)
	if err == nil {
		IsInstallerOutdated = res.TagName != InstallerTag
	}
}

func GetInstallerDownloadLink() string {
	switch runtime.GOOS {
	case "windows":
		return "https://github.com/Vencord/Installer/releases/latest/download/VencordInstaller.exe"
	case "darwin":
		return "https://github.com/Vencord/Installer/releases/latest/download/VencordInstaller.MacOS.zip"
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
