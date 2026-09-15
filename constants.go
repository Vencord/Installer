/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"image/color"
	"vencordinstaller/buildinfo"
)

const ReleaseUrl = "https://api.github.com/repos/Vendicated/Vencord/releases/latest"
const ReleaseUrlFallback = "https://vencord.dev/releases/vencord"
const InstallerReleaseUrl = "https://api.github.com/repos/Vencord/Installer/releases/latest"
const InstallerReleaseUrlFallback = "https://vencord.dev/releases/installer"

var UserAgent = "VencordInstaller/" + buildinfo.InstallerGitHash + " (https://github.com/Vencord/Installer)"

var (
	DiscordGreen        = color.RGBA{R: 0, G: 133, B: 69, A: 0xff}
	DiscordGreenHovered = color.RGBA{R: 0, G: 108, B: 55, A: 0xff}
	DiscordRed          = color.RGBA{R: 210, G: 45, B: 57, A: 0xff}
	DiscordRedHovered   = color.RGBA{R: 169, G: 35, B: 46, A: 0xff}
	DiscordBlue         = color.RGBA{R: 88, G: 101, B: 242, A: 0xff}
	DiscordBlueHovered  = color.RGBA{R: 68, G: 82, B: 187, A: 0xff}
	DiscordYellow       = color.RGBA{R: 0xfe, G: 0xe7, B: 0x5c, A: 0xff}
)

var LinuxDiscordNames = []string{
	"Discord",
	"DiscordPTB",
	"DiscordCanary",
	"DiscordDevelopment",
	"discord",
	"discordptb",
	"discordcanary",
	"discorddevelopment",
	"discord-ptb",
	"discord-canary",
	"discord-development",
	// Flatpak
	"com.discordapp.Discord",
	"com.discordapp.DiscordPTB",
	"com.discordapp.DiscordCanary",
	"com.discordapp.DiscordDevelopment",
}
