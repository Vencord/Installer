/*
 * This part is file of VencordInstaller
 * Copyright (c) 2022 Vendicated
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
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
	fmt.Println("Checking if", path, "exists:", Ternary(err == nil, "Yes", "No"))
	return err == nil
}

func IsDirectory(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		fmt.Println("Error while checking if", path, "is directory:", err)
		return false
	}
	fmt.Println("Checking if", path, "is directory:", Ternary(s.IsDir(), "Yes", "No"))
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
