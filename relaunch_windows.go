/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2026 Vendicated and Vencord contributors
 */

package main

import (
	"os"
	"os/exec"
	path "path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const processCommandLineInformation = 60

func queryProcessImagePath(handle windows.Handle) (string, error) {
	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(handle, 0, &buf[0], &size); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}

func queryProcessCommandLine(handle windows.Handle) (string, error) {
	buf := make([]byte, 4096)
	var retLen uint32

	err := windows.NtQueryInformationProcess(
		handle,
		processCommandLineInformation,
		unsafe.Pointer(&buf[0]),
		uint32(len(buf)),
		&retLen,
	)
	if err != nil && retLen > uint32(len(buf)) {
		buf = make([]byte, retLen)
		err = windows.NtQueryInformationProcess(
			handle,
			processCommandLineInformation,
			unsafe.Pointer(&buf[0]),
			uint32(len(buf)),
			&retLen,
		)
	}
	if err != nil {
		return "", err
	}

	ustr := (*windows.NTUnicodeString)(unsafe.Pointer(&buf[0]))
	if ustr.Buffer == nil || ustr.Length == 0 {
		runtime.KeepAlive(buf)
		return "", nil
	}

	cmdLine := ustr.String()
	runtime.KeepAlive(buf)
	return cmdLine, nil
}

func parseWindowsCommandLine(cmdLine string) []string {
	if strings.TrimSpace(cmdLine) == "" {
		return nil
	}

	ptr, err := windows.UTF16PtrFromString(cmdLine)
	if err != nil {
		return nil
	}

	var argc int32
	argv, err := windows.CommandLineToArgv(ptr, &argc)
	if err != nil {
		return nil
	}
	defer func() {
		_, _ = windows.LocalFree(windows.Handle(unsafe.Pointer(argv)))
	}()

	args := make([]string, 0, argc)
	for i := int32(0); i < argc; i++ {
		args = append(args, windows.UTF16ToString(argv[i][:]))
	}
	return args
}

func (di *DiscordInstall) findRunningInstance() *RunningDiscord {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer func() {
		_ = windows.CloseHandle(snapshot)
	}()

	resolvedPath := resolvePath(di.path)
	selfPid := uint32(os.Getpid())

	var result RunningDiscord
	entry := windows.ProcessEntry32{
		Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{})),
	}

	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		pid := entry.ProcessID
		if pid == 0 || pid == selfPid {
			continue
		}

		handle, openErr := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
		if openErr != nil {
			continue
		}

		exePath, pathErr := queryProcessImagePath(handle)
		if pathErr == nil && (isPathWithinDir(exePath, resolvedPath, true) || isPathWithinDir(exePath, di.path, true)) {
			result.pids = append(result.pids, int(pid))

			if result.exe == "" && !isIgnoredHelperBinary(exePath) {
				var args []string
				if cmdLine, cmdErr := queryProcessCommandLine(handle); cmdErr == nil {
					if parsed := parseWindowsCommandLine(cmdLine); len(parsed) > 1 {
						args = parsed[1:]
					}
				}
				if !isElectronHelperArgv(args) {
					result.pid = int(pid)
					result.exe = exePath
					result.args = args
				}
			}
		}

		_ = windows.CloseHandle(handle)
	}

	if len(result.pids) == 0 {
		return nil
	}
	return &result
}

func killRunningInstance(_ *DiscordInstall, info *RunningDiscord) error {
	var handles []windows.Handle

	for _, pid := range info.pids {
		handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, uint32(pid))
		if err != nil {
			continue
		}
		if err := windows.TerminateProcess(handle, 1); err != nil {
			_ = windows.CloseHandle(handle)
			continue
		}
		handles = append(handles, handle)
	}

	deadline := time.Now().Add(5 * time.Second)
	for _, handle := range handles {
		if remaining := time.Until(deadline); remaining > 0 {
			_, _ = windows.WaitForSingleObject(handle, uint32(remaining.Milliseconds()))
		}
		_ = windows.CloseHandle(handle)
	}

	return nil
}

func launchDetached(_ *DiscordInstall, info *RunningDiscord) error {
	Log.Info("Relaunching", info.exe, "with args", info.args)

	cmd := exec.Command(info.exe, info.args...)
	cmd.Dir = path.Dir(info.exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
