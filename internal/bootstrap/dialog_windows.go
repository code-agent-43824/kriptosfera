//go:build windows

package bootstrap

import (
	"syscall"
	"unsafe"
)

const mbIconError = 0x00000010

func showLauncherErrorDialog(title, text string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBoxW := user32.NewProc("MessageBoxW")
	messageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		mbIconError,
	)
}
