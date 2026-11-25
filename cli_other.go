//go:build !windows

package main

import (
	"os"

	"github.com/fatih/color"
	"golang.org/x/sys/unix"
)

func IsDoubleClickRun() bool {
	return false
}

func getTerminalSize() (int, int) {
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
		if err == nil && ws.Col > 0 && ws.Row > 0 {
			return int(ws.Col), int(ws.Row)
		}
	}
	return 80, 24
}

func initAnsiColor() bool {
	return !color.NoColor
}
