//go:build gui || (!gui && !cli)

package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

func init() {
	if IsAdmin() {
		const (
			MB_OKCANCEL        = 0x1
			MB_ICONEXCLAMATION = 0x30

			IDCANCEL = 2
		)

		ret, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(
			uintptr(0), // hwnd (NULL)
			uintptr(unsafe.Pointer(Unwrap(syscall.UTF16PtrFromString(
				"Run me as a normal user, not administrator!\n"+
					"If you didn't explicitly run me as Administrator, make sure you don't have UAC set to 'Never Notify'.\n\n"+
					"VencordInstaller will close once you press OK.\n"+
					"Alternatively, press Cancel to proceed anyway, but this may cause issues. Only choose this option as a last resort",
			)))),
			uintptr(unsafe.Pointer(Unwrap(syscall.UTF16PtrFromString("Do not run me as Administrator")))),
			uintptr(MB_OKCANCEL|MB_ICONEXCLAMATION), // flags
		)

		if ret != IDCANCEL {
			panic("Ran as Administrator")
		}
	}
}

func IsAdmin() bool {
	// most sane windows code, copy-pasted from https://github.com/golang/go/issues/28804#issuecomment-505326268

	var sid *windows.SID

	// Although this looks scary, it is directly copied from the
	// official windows documentation. The Go API for this is a
	// direct wrap around the official C++ API.
	// See https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		fmt.Println("SID Error: ", err)
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)

	isMember, err := token.IsMember(sid)
	if err != nil {
		fmt.Println("Token Membership Error: ", err)
		return false
	}

	return token.IsElevated() || isMember
}
