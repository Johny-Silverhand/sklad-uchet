//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jchv/go-webview2"
)

// runNativeWindow opens an embedded WebView2 window (no browser chrome).
// Blocks until the window is closed. Returns true if the native shell was used.
func runNativeWindow(url, mode string) bool {
	w, h := 1120, 740
	title := appName + " · Установка"
	if mode == "app" {
		w, h = 1360, 860
		title = appName
	} else if mode == "uninstall" {
		w, h = 720, 520
		title = appName + " · Удаление"
	}

	data, err := dataDir()
	if err != nil {
		data = os.TempDir()
	}
	userData := filepath.Join(data, "webview2-profile")
	_ = os.MkdirAll(userData, 0o755)

	wv := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		DataPath:  userData,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  title,
			Width:  uint(w),
			Height: uint(h),
			Center: true,
		},
	})
	if wv == nil {
		fmt.Fprintln(os.Stderr, "WebView2 недоступен — нужен Microsoft Edge WebView2 Runtime")
		return false
	}
	defer wv.Destroy()
	wv.SetSize(w, h, webview2.HintNone)
	wv.Navigate(url)
	wv.Run()
	return true
}
