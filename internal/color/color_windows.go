//go:build windows

package color

import (
	"os"

	"golang.org/x/sys/windows"
)

func init() {
	enableVT(os.Stdout)
	enableVT(os.Stderr)
}

func enableVT(f *os.File) {
	if f == nil {
		return
	}
	var mode uint32
	h := windows.Handle(f.Fd())
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return
	}
	mode |= windows.ENABLE_PROCESSED_OUTPUT | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	_ = windows.SetConsoleMode(h, mode)
}
