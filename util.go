/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"syscall"
)

func ArrayIncludes[T comparable](arr []T, v T) bool {
	for _, e := range arr {
		if e == v {
			return true
		}
	}
	return false
}

func ExistsFile(path string) bool {
	_, err := os.Stat(path)
	Log.Debug("Checking if", path, "exists:", Ternary(err == nil, "Yes", "No"))
	return err == nil
}

func IsDirectory(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		Log.Error("Error while checking if", path, "is directory:", err)
		return false
	}
	Log.Debug("Checking if", path, "is directory:", Ternary(s.IsDir(), "Yes", "No"))
	return s.IsDir()
}

func Ternary[T any](b bool, ifTrue, ifFalse T) T {
	if b {
		return ifTrue
	}
	return ifFalse
}

var branches = []string{"canary", "development", "ptb"}

func GetBranch(name string) string {
	name = strings.ToLower(name)
	for _, branch := range branches {
		if strings.HasSuffix(name, branch) {
			return branch
		}
	}
	return "stable"
}

func Ptr[T any](v T) *T {
	return &v
}

func CheckIfErrIsCauseItsBusyRn(err error) error {
	if runtime.GOOS != "windows" {
		return err
	}

	// bruhhhh
	if linkError, ok := err.(*os.LinkError); ok {
		if errno, ok := linkError.Err.(syscall.Errno); ok && errno == 32 /* ERROR_SHARING_VIOLATION */ {
			return errors.New(
				"Cannot patch because Discord's files are used by a different process." +
					"\nMake sure you close Discord before trying to patch!",
			)
		}
	}

	return err
}

func Prepend[T any](slice []T, elems ...T) []T {
	return append(elems, slice...)
}
