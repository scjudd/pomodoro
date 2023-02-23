//go:build darwin || freebsd || netbsd || openbsd || solaris || dragonfly
// +build darwin freebsd netbsd openbsd solaris dragonfly

package main

import "syscall"

const ioctlReadTermios = syscall.TIOCGETA
const ioctlWriteTermios = syscall.TIOCSETA
