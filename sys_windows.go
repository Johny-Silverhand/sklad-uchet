//go:build windows

package main

import "syscall"

func hideWindow() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}

func showWindow() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: false}
}
