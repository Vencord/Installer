package main

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/fatih/color"
	"golang.org/x/sys/windows"
)

func IsDoubleClickRun() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	lp := kernel32.NewProc("GetConsoleProcessList")
	if lp != nil {
		var pids [2]uint32
		var maxCount uint32 = 2
		ret, _, _ := lp.Call(uintptr(unsafe.Pointer(&pids)), uintptr(maxCount))
		if ret > 1 {
			return false
		}
	}
	return true
}

func getTerminalSize() (int, int) {
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		var info windows.ConsoleScreenBufferInfo
		err := windows.GetConsoleScreenBufferInfo(windows.Handle(f.Fd()), &info)
		if err == nil {
			width := int(info.Window.Right - info.Window.Left + 1)
			height := int(info.Window.Bottom - info.Window.Top + 1)
			if width > 0 && height > 0 {
				return width, height
			}
		}
	}
	return 80, 24
}

func enableVirtualTerminal(f *os.File) bool {
	handle := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return false
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}
	return windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}

func initAnsiColor() bool {
	if color.NoColor {
		return false
	}

	stdoutOk := enableVirtualTerminal(os.Stdout)
	_ = enableVirtualTerminal(os.Stderr)

	if stdoutOk {
		return true
	}

	if os.Getenv("WT_SESSION") != "" ||
		os.Getenv("ANSICON") != "" ||
		os.Getenv("ConEmuANSI") == "ON" ||
		os.Getenv("TERM_PROGRAM") != "" {
		return true
	}

	color.NoColor = true
	return false
}
