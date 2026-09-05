//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func hideWindow() *syscall.SysProcAttr { return nil }
func showWindow() *syscall.SysProcAttr { return nil }

func showError(title, msg string) {
	fmt.Fprintln(os.Stderr, title+":", msg)
}

func showInfo(title, msg string) {
	fmt.Fprintln(os.Stderr, title+":", msg)
}
