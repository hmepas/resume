//go:build linux

package tui

import "syscall"

const (
	ioctlGetTermios = syscall.TCGETS
	ioctlSetTermios = syscall.TCSETS
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
