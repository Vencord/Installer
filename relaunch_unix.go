//go:build linux || darwin

/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2026 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"syscall"
	"time"
)

func waitForPidExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func killUnixProcess(pid int) {
	if pid <= 0 {
		return
	}

	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if !errors.Is(err, syscall.ESRCH) {
			Log.Debug("Failed to send SIGTERM to pid", pid, ":", err)
		}
		return
	}

	if waitForPidExit(pid, 5*time.Second) {
		return
	}

	Log.Debug("Force-killing Discord pid", pid)
	_ = syscall.Kill(pid, syscall.SIGKILL)
	waitForPidExit(pid, 2*time.Second)
}
