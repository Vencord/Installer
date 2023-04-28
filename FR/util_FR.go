/*
 * SPDX-License-Identifier: GPL-3.0
 * Vencord Installer, a cross platform gui/cli app for installing Vencord
 * Copyright (c) 2023 Vendicated and Vencord contributors
 */

package main

import (
	"errors"
	"fmt"
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
	fmt.Println("On vérifie si ", path, "existe:", Ternary(err == nil, "Oui", "Non"))
	return err == nil
}

func IsDirectory(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		fmt.Println("Erreur lors de la vérification du dossier", path, "est un dossier:", err)
		return false
	}
	fmt.Println("On vérifie si ", path, "est un dossier", Ternary(s.IsDir(), "Oui", "Non"))
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
				"Impossible de patcher Discord car il est ouvert!" +
					"\nMerci de fermer Discord depuis le gestionnaire des tâches et réessayer",
			)
		}
	}

	return err
}
