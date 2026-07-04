//go:build darwin

package tui

import "syscall"

const (
	ioctlGetTermios = syscall.TIOCGETA
	ioctlSetTermios = syscall.TIOCSETA
	ioctlGetWinsize = syscall.TIOCGWINSZ
	termiosVMIN     = syscall.VMIN
	termiosVTIME    = syscall.VTIME
)

type winsize struct {
	row    uint16
	col    uint16
	xpixel uint16
	ypixel uint16
}
