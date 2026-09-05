package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

func openAppWindow(url, mode string) (*exec.Cmd, error) {
	w, h := 1120, 740
	if mode == "app" {
		w, h = 1360, 860
	}
	profileDir, err := dataDir()
	if err != nil {
		profileDir = os.TempDir()
	}
	profile := filepath.Join(profileDir, "chromium-profile")
	_ = os.MkdirAll(profile, 0o755)

	args := []string{
		"--app=" + url,
		"--window-size=" + strconv.Itoa(w) + "," + strconv.Itoa(h),
		"--user-data-dir=" + profile,
		"--no-first-run",
		"--disable-extensions",
		"--disable-default-apps",
	}

	if runtime.GOOS == "windows" {
		candidates := []string{
			filepath.Join(os.Getenv("PROGRAMFILES(X86)"), `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(os.Getenv("PROGRAMFILES"), `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(os.Getenv("ProgramFiles"), `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(os.Getenv("PROGRAMFILES"), `Google\Chrome\Application\chrome.exe`),
			filepath.Join(os.Getenv("LOCALAPPDATA"), `Google\Chrome\Application\chrome.exe`),
		}
		for _, bin := range candidates {
			if st, err := os.Stat(bin); err == nil && !st.IsDir() {
				cmd := exec.Command(bin, args...)
				cmd.SysProcAttr = hideWindow()
				if err := cmd.Start(); err != nil {
					return nil, err
				}
				return cmd, nil
			}
		}
		cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		cmd.SysProcAttr = hideWindow()
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	linuxBins := []string{"microsoft-edge", "microsoft-edge-stable", "google-chrome", "chromium", "chromium-browser"}
	if runtime.GOOS == "darwin" {
		macBins := []string{
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}
		for _, bin := range macBins {
			if st, err := os.Stat(bin); err == nil && !st.IsDir() {
				cmd := exec.Command(bin, args...)
				if err := cmd.Start(); err != nil {
					return nil, err
				}
				return cmd, nil
			}
		}
	}
	for _, bin := range linuxBins {
		if path, err := exec.LookPath(bin); err == nil {
			cmd := exec.Command(path, args...)
			if err := cmd.Start(); err != nil {
				continue
			}
			return cmd, nil
		}
	}
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", url).Start()
	} else {
		_ = exec.Command("xdg-open", url).Start()
	}
	return nil, nil
}
