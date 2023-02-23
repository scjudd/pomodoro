package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var origTermios syscall.Termios

func enterRawTerminalMode() {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, 0, ioctlReadTermios, uintptr(unsafe.Pointer(&origTermios)))
	if err != 0 {
		panic("couldn't get current terminal configuration")
	}

	t := origTermios
	t.Lflag &^= syscall.ECHO | syscall.ICANON

	_, _, err = syscall.Syscall(syscall.SYS_IOCTL, 0, ioctlWriteTermios, uintptr(unsafe.Pointer(&t)))
	if err != 0 {
		panic("couldn't update terminal configuration")
	}
}

func restoreTerminalMode() {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, 0, ioctlWriteTermios, uintptr(unsafe.Pointer(&origTermios)))
	if err != 0 {
		panic("couldn't restore terminal configuration")
	}
}

func getWindowSize() (rows, cols int) {
	winsize := struct {
		rows uint16
		cols uint16
		_    uint16
		_    uint16
	}{}

	// See usage #4 of unsafe.Pointer here: https://pkg.go.dev/unsafe#Pointer.
	// It is very important that the uintptr(unsafe.Pointer(...))
	// conversion appear in the argument list to syscall.Syscall.
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, 0, syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&winsize)))
	if err != 0 {
		panic("couldn't get window size")
	}

	return int(winsize.rows), int(winsize.cols)
}

const (
	escClearDisplay                  = "\x1b[2J"
	escClearLine                     = "\x1b[2K"
	escCursorHide                    = "\x1b[?25l"
	escCursorShow                    = "\x1b[?25h"
	escSgrForegroundRed              = "\x1b[91m"
	escSgrReset                      = "\x1b[0m"
	escSgrReverseVideo               = "\x1b[7m"
	escXtermAlternativeScreenDisable = "\x1b[?1049l"
	escXtermAlternativeScreenEnable  = "\x1b[?1049h"
)

func escCursorMove(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row, col)
}
