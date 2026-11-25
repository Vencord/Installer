/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2026 Vendicated and Vencord contributors
 */

package main

import (
	"bytes"
	"os"
	"os/exec"
	"os/user"
	path "path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func readCmdline(procDir string) []string {
	data, err := os.ReadFile(path.Join(procDir, "cmdline"))
	if err != nil {
		return nil
	}

	data = bytes.TrimRight(data, "\x00")
	if len(data) == 0 {
		return nil
	}

	parts := bytes.Split(data, []byte{0})
	args := make([]string, len(parts))
	for i, part := range parts {
		args[i] = string(part)
	}
	return args
}

func readFlatpakID(procDir string) string {
	data, err := os.ReadFile(path.Join(procDir, "root", ".flatpak-info"))
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "name=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "name="))
		}
	}
	return ""
}

func sudoCredential() *syscall.Credential {
	if os.Geteuid() != 0 {
		return nil
	}

	u, err := user.Lookup(os.Getenv("SUDO_USER"))
	if err != nil {
		return nil
	}

	uid, err1 := strconv.Atoi(u.Uid)
	gid, err2 := strconv.Atoi(u.Gid)
	if err1 != nil || err2 != nil {
		return nil
	}

	return &syscall.Credential{
		Uid: uint32(uid),
		Gid: uint32(gid),
	}
}

func (di *DiscordInstall) findRunningInstance() *RunningDiscord {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	selfPid := os.Getpid()
	resolvedPath := resolvePath(di.path)
	flatpakID := ""
	if di.isFlatpak {
		flatpakID = getFlatpakAppId(di.path)
		if flatpakID == "" {
			return nil
		}
	}

	var fallback *RunningDiscord

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || pid == selfPid {
			continue
		}

		procDir := path.Join("/proc", entry.Name())
		cmdline := readCmdline(procDir)
		if len(cmdline) == 0 {
			continue
		}

		if di.isFlatpak {
			if readFlatpakID(procDir) != flatpakID {
				continue
			}

			// Flatpak wrapper script holds the exact user-passed args
			wrapper := "/app/bin/" + flatpakID
			for i, arg := range cmdline {
				if arg == wrapper {
					return &RunningDiscord{
						pid:     pid,
						flatpak: flatpakID,
						args:    cmdline[i+1:],
					}
				}
			}

			if fallback == nil && !isElectronHelperArgv(cmdline[1:]) {
				base := strings.ToLower(path.Base(cmdline[0]))
				if strings.Contains(base, "discord") {
					fallback = &RunningDiscord{
						pid:     pid,
						flatpak: flatpakID,
						args:    cmdline[1:],
					}
				}
			}
			continue
		}

		exe, err := os.Readlink(path.Join(procDir, "exe"))
		if err != nil {
			continue
		}
		exe = strings.TrimSuffix(exe, " (deleted)")

		if !isPathWithinDir(exe, resolvedPath, false) && !isPathWithinDir(exe, di.path, false) {
			continue
		}
		if isIgnoredHelperBinary(exe) || isElectronHelperArgv(cmdline[1:]) {
			continue
		}
		if readFlatpakID(procDir) != "" {
			continue
		}

		return &RunningDiscord{
			pid:  pid,
			exe:  exe,
			args: cmdline[1:],
		}
	}

	return fallback
}

func killRunningInstance(_ *DiscordInstall, info *RunningDiscord) error {
	if info.flatpak != "" {
		killCmd := exec.Command("flatpak", "kill", info.flatpak)
		killCmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: sudoCredential(),
		}
		if err := killCmd.Run(); err == nil {
			waitForPidExit(info.pid, 5*time.Second)
			return nil
		}
	}

	killUnixProcess(info.pid)
	return nil
}

func launchDetached(_ *DiscordInstall, info *RunningDiscord) error {
	var cmd *exec.Cmd
	if info.flatpak != "" {
		args := append([]string{"run", info.flatpak}, info.args...)
		Log.Info("Relaunching Flatpak", info.flatpak, "with args", info.args)
		cmd = exec.Command("flatpak", args...)
	} else {
		Log.Info("Relaunching", info.exe, "with args", info.args)
		cmd = exec.Command(info.exe, info.args...)
		cmd.Dir = path.Dir(info.exe)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:     true,
		Credential: sudoCredential(),
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}
