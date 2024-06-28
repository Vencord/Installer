/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	path "path/filepath"
	"strings"
	"sync"
	"unsafe"
)

var windowsNames = map[string]string{
	"stable": "Discord",
	"ptb":    "DiscordPTB",
	"canary": "DiscordCanary",
	"dev":    "DiscordDevelopment",
}

var killLock sync.Mutex

func ParseDiscord(p, branch string) *DiscordInstall {
	entries, err := os.ReadDir(p)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			Log.Warn("Error during readdir "+p+":", err)
		}
		return nil
	}

	isPatched := false
	appPath := ""
	for _, dir := range entries {
		if dir.IsDir() && strings.HasPrefix(dir.Name(), "app-") {
			resources := path.Join(p, dir.Name(), "resources")
			if !ExistsFile(resources) {
				continue
			}
			app := path.Join(resources, "app")
			if app > appPath {
				appPath = app
				isPatched = ExistsFile(path.Join(resources, "_app.asar"))
			}
		}
	}

	if appPath == "" {
		return nil
	}

	if branch == "" {
		branch = GetBranch(p)
	}

	return &DiscordInstall{
		path:             p,
		branch:           branch,
		appPath:          appPath,
		isPatched:        isPatched,
		isFlatpak:        false,
		isSystemElectron: false,
	}
}

func FindDiscords() []any {
	var discords []any

	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		Log.Error("%LOCALAPPDATA% is empty???????")
		return discords
	}

	for branch, dirname := range windowsNames {
		p := path.Join(appData, dirname)
		if discord := ParseDiscord(p, branch); discord != nil {
			Log.Debug("Found Discord install at ", p)
			discords = append(discords, discord)
		}
	}
	return discords
}

func KillDiscord(di *DiscordInstall) bool {
	killLock.Lock()
	defer killLock.Unlock()
	
	name := windowsNames[di.branch]
	Log.Debug("Trying to kill", name)
	pid := findProcessIdByName(name + ".exe")
	if pid == 0 {
		Log.Debug("Didn't find process matching name")
		return false
	}

	proc, err := os.FindProcess(int(pid))
	if err != nil {
		Log.Warn("Failed to find process with pid", pid)
		return false
	}

	err = proc.Kill()
	if err != nil {
		Log.Warn("Failed to kill", name+":", err)
		return false
	} else {
		Log.Debug("Waiting for", name, "to exit")
		_, _ = proc.Wait()
		return true
	}
}

func FixOwnership(_ string) error {
	return nil
}

// https://github.com/Vencord/Installer/issues/9

func CheckScuffedInstall() bool {
	username := os.Getenv("USERNAME")
	programData := os.Getenv("PROGRAMDATA")
	for _, discordName := range windowsNames {
		if ExistsFile(path.Join(programData, username, discordName)) || ExistsFile(path.Join(programData, username, discordName)) {
			HandleScuffedInstall()
			return true
		}
	}
	return false
}

func findProcessIdByName(name string) uint32 {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}

	procEntry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for {
		err = windows.Process32Next(snapshot, &procEntry)
		if err != nil {
			return 0
		}
		if windows.UTF16ToString(procEntry.ExeFile[:]) == name {
			return procEntry.ProcessID
		}
	}
}
