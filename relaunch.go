/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2026 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"os"
	path "path/filepath"
	"strings"
)

type RunningDiscord struct {
	pid     int
	pids    []int
	exe     string
	args    []string
	flatpak string
}

func resolvePath(p string) string {
	cleaned := path.Clean(p)
	if resolved, err := path.EvalSymlinks(cleaned); err == nil {
		return path.Clean(resolved)
	}
	return cleaned
}

func isPathWithinDir(target, dir string, caseInsensitive bool) bool {
	if target == "" || dir == "" {
		return false
	}

	t := path.Clean(target)
	d := path.Clean(dir)
	if caseInsensitive {
		t = strings.ToLower(t)
		d = strings.ToLower(d)
	}
	if t == d {
		return true
	}

	sep := string(os.PathSeparator)
	if !strings.HasSuffix(d, sep) {
		d += sep
	}
	return strings.HasPrefix(t, d)
}

func isElectronHelperArgv(args []string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--type=") || arg == "--type" {
			return true
		}
	}
	return false
}

func isIgnoredHelperBinary(exePath string) bool {
	switch strings.ToLower(path.Base(exePath)) {
	case "update.exe", "squirrel.exe", "crashpad_handler", "crashpad_handler.exe", "chrome_crashpad_handler", "chrome-sandbox":
		return true
	}
	return false
}

func getFlatpakAppId(p string) string {
	for _, segment := range strings.Split(path.ToSlash(p), "/") {
		if strings.HasPrefix(segment, "com.discordapp.") {
			return segment
		}
	}
	return ""
}

func (di *DiscordInstall) FindRunning() *RunningDiscord {
	return di.findRunningInstance()
}

func (di *DiscordInstall) ClearPendingRelaunch() {
	di.relaunchInfo = nil
}

func (di *DiscordInstall) RelaunchKilled() (bool, error) {
	info := di.relaunchInfo
	di.relaunchInfo = nil
	if info == nil || (info.exe == "" && info.flatpak == "") {
		return false, nil
	}
	return true, launchDetached(di, info)
}

func (di *DiscordInstall) Relaunch(fallback *RunningDiscord) error {
	info := di.findRunningInstance()
	if info != nil {
		if err := killRunningInstance(di, info); err != nil {
			return err
		}
		if info.exe != "" || info.flatpak != "" {
			return launchDetached(di, info)
		}
	}

	if fallback != nil && (fallback.exe != "" || fallback.flatpak != "") {
		return launchDetached(di, fallback)
	}

	return errors.New("Could not find a running Discord process to relaunch")
}
