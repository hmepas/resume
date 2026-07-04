//go:build darwin || linux

package tui

import (
	"os"
	"syscall"
	"unsafe"
)

type rawState struct {
	fd  int
	old syscall.Termios
}

func isTerminal(file *os.File) bool {
	_, err := getSize(file.Fd())
	return err == nil
}

func makeRaw(file *os.File) (*rawState, error) {
	fd := int(file.Fd())
	var old syscall.Termios
	if err := ioctl(fd, ioctlGetTermios, uintptr(unsafe.Pointer(&old))); err != nil {
		return nil, err
	}

	raw := old
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cc[termiosVMIN] = 1
	raw.Cc[termiosVTIME] = 0

	if err := ioctl(fd, ioctlSetTermios, uintptr(unsafe.Pointer(&raw))); err != nil {
		return nil, err
	}
	return &rawState{fd: fd, old: old}, nil
}

func (s *rawState) restore() {
	_ = ioctl(s.fd, ioctlSetTermios, uintptr(unsafe.Pointer(&s.old)))
}

func terminalSize(file *os.File) (int, int) {
	size, err := getSize(file.Fd())
	if err != nil || size.col == 0 || size.row == 0 {
		return 80, 24
	}
	return int(size.col), int(size.row)
}

func getSize(fd uintptr) (winsize, error) {
	var size winsize
	err := ioctl(int(fd), ioctlGetWinsize, uintptr(unsafe.Pointer(&size)))
	return size, err
}

func ioctl(fd int, req uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}
