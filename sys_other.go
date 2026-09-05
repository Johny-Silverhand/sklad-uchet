//go:build !windows

package main

import "syscall"

func hideWindow() *syscall.SysProcAttr { return nil }
func showWindow() *syscall.SysProcAttr { return nil }
