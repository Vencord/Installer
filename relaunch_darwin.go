/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2026 Vendicated and Vencord contributors
 */

package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	path "path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func getMacProcessArgs(pid int) (string, []string, bool) {
	buf, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil || len(buf) < 4 {
		return "", nil, false
	}

	argc := int(int32(binary.LittleEndian.Uint32(buf[:4])))
	if argc < 0 {
		return "", nil, false
	}

	pos := 4
	nul := bytes.IndexByte(buf[pos:], 0)
	if nul < 0 {
		return "", nil, false
	}

	execPath := string(buf[pos : pos+nul])
	pos += nul + 1

	for pos < len(buf) && buf[pos] == 0 {
		pos++
	}

	argv := make([]string, 0, argc)
	for i := 0; i < argc && pos < len(buf); i++ {
		end := bytes.IndexByte(buf[pos:], 0)
		if end < 0 {
			argv = append(argv, string(buf[pos:]))
			break
		}
		argv = append(argv, string(buf[pos:pos+end]))
		pos += end + 1
	}

	return execPath, argv, true
}

func (di *DiscordInstall) findRunningInstance() *RunningDiscord {
	procs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil
	}

	macosDir := path.Join(resolvePath(di.path), "Contents", "MacOS")
	rawMacosDir := path.Join(di.path, "Contents", "MacOS")
	selfPid := os.Getpid()

	for _, kp := range procs {
		pid := int(kp.Proc.P_pid)
		if pid <= 0 || pid == selfPid {
			continue
		}

		exePath, argv, ok := getMacProcessArgs(pid)
		if !ok || exePath == "" {
			continue
		}

		if !isPathWithinDir(exePath, macosDir, false) && !isPathWithinDir(exePath, rawMacosDir, false) {
			continue
		}

		var args []string
		if len(argv) > 1 {
			for _, arg := range argv[1:] {
				if !strings.HasPrefix(arg, "-psn_") {
					args = append(args, arg)
				}
			}
		}
		if isElectronHelperArgv(args) {
			continue
		}

		return &RunningDiscord{
			pid:  pid,
			exe:  exePath,
			args: args,
		}
	}

	return nil
}

func killRunningInstance(_ *DiscordInstall, info *RunningDiscord) error {
	killUnixProcess(info.pid)
	return nil
}

func launchDetached(di *DiscordInstall, info *RunningDiscord) error {
	if strings.HasSuffix(di.path, ".app") {
		openArgs := []string{"-n", "-a", di.path}
		if len(info.args) > 0 {
			openArgs = append(openArgs, "--args")
			openArgs = append(openArgs, info.args...)
		}

		Log.Info("Relaunching", di.path, "with args", info.args)
		if err := exec.Command("open", openArgs...).Run(); err == nil {
			return nil
		}
	}

	Log.Info("Relaunching", info.exe, "with args", info.args)
	cmd := exec.Command(info.exe, info.args...)
	cmd.Dir = path.Dir(info.exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}
