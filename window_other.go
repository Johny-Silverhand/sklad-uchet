//go:build !windows

package main

// On non-Windows builds, native WebView2 is unavailable.
func runNativeWindow(url, mode string) bool {
	return false
}
