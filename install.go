package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type installReq struct {
	Dir       string `json:"dir"`
	Desktop   bool   `json:"desktop"`
	StartMenu bool   `json:"startMenu"`
	Autostart bool   `json:"autostart"`
}

func attachInstallerAPI(mux *http.ServeMux, mode string) {
	mux.HandleFunc("/api/meta", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"native":     true,
			"app":        appName,
			"version":    appVersion,
			"publisher":  publisher,
			"credit":     creditLine,
			"mode":       mode,
			"defaultDir": defaultInstallDir(),
		})
	})
	mux.HandleFunc("/api/install", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		var req installReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Некорректный запрос"})
			return
		}
		if err := doInstall(req); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "credit": creditLine})
	})
	mux.HandleFunc("/api/finish", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Launch bool `json:"launch"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{"ok": true})
		if body.Launch {
			go func() {
				time.Sleep(400 * time.Millisecond)
				exe, _ := os.Executable()
				target := exe
				if b, err := os.ReadFile(markerPath()); err == nil {
					var m map[string]string
					if json.Unmarshal(b, &m) == nil && m["exe"] != "" {
						target = m["exe"]
					}
				}
				cmd := exec.Command(target, "--app")
				cmd.Dir = filepath.Dir(target)
				_ = cmd.Start()
				time.Sleep(300 * time.Millisecond)
				os.Exit(0)
			}()
		}
	})
	mux.HandleFunc("/api/browse", func(w http.ResponseWriter, r *http.Request) {
		dir, err := pickFolder()
		if err != nil || strings.TrimSpace(dir) == "" {
			writeJSON(w, 200, map[string]string{"dir": defaultInstallDir()})
			return
		}
		writeJSON(w, 200, map[string]string{"dir": strings.TrimSpace(dir)})
	})
	mux.HandleFunc("/api/uninstall", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		go func() {
			time.Sleep(350 * time.Millisecond)
			runUninstall()
			os.Exit(0)
		}()
		writeJSON(w, 200, map[string]any{"ok": true, "credit": creditLine})
	})
}

func defaultInstallDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, "AppData", "Local")
	}
	if base == "" {
		base = `C:\Program Files`
	}
	return filepath.Join(base, "Programs", publisher, appExeName)
}

func pickFolder() (string, error) {
	if runtime.GOOS != "windows" {
		return defaultInstallDir(), nil
	}
	ps := `Add-Type -AssemblyName System.Windows.Forms; $d = New-Object System.Windows.Forms.FolderBrowserDialog; $d.Description = 'Папка установки Склад Учёт'; $d.ShowNewFolderButton = $true; if ($d.ShowDialog() -eq 'OK') { $d.SelectedPath }`
	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-Command", ps)
	cmd.SysProcAttr = showWindow()
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func markerPath() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(base, publisher, "sklad-uchet-install.json")
}

func doInstall(req installReq) error {
	dir := strings.TrimSpace(req.Dir)
	if dir == "" {
		dir = defaultInstallDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	src, err := os.Executable()
	if err != nil {
		return err
	}
	destExe := filepath.Join(dir, appExeName+".exe")
	if err := copyFile(src, destExe); err != nil {
		return err
	}
	if b, err := webFS.ReadFile("web/setup/media/app.ico"); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "app.ico"), b, 0o644)
	}
	_ = os.WriteFile(filepath.Join(dir, "LICENSE.txt"), []byte(licenseText()), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "installed.json"), []byte(`{"ok":true,"app":"`+appExeName+`","version":"`+appVersion+`"}`), 0o644)
	uninst := filepath.Join(dir, "Uninstall.exe")
	_ = copyFile(src, uninst)
	marker, _ := json.Marshal(map[string]string{"exe": destExe, "dir": dir})
	_ = os.MkdirAll(filepath.Dir(markerPath()), 0o755)
	_ = os.WriteFile(markerPath(), marker, 0o644)

	icon := filepath.Join(dir, "app.ico")
	if req.StartMenu {
		programs := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", publisher)
		_ = os.MkdirAll(programs, 0o755)
		_ = writeShortcut(filepath.Join(programs, appName+".lnk"), destExe, dir, icon, "--app")
		_ = writeShortcut(filepath.Join(programs, "Удалить "+appName+".lnk"), uninst, dir, icon, "--uninstall")
	}
	if req.Desktop {
		desk := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", appName+".lnk")
		_ = writeShortcut(desk, destExe, dir, icon, "--app")
	}
	if req.Autostart {
		run := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup", appName+".lnk")
		_ = writeShortcut(run, destExe, dir, icon, "--app")
	}
	writeUninstallReg(dir, destExe, uninst)
	return nil
}

func writeShortcut(lnk, target, workdir, icon string, args ...string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	arg := strings.Join(args, " ")
	ps := `$s = New-Object -ComObject WScript.Shell; $l = $s.CreateShortcut('` + psQuote(lnk) + `'); $l.TargetPath = '` + psQuote(target) + `'; $l.WorkingDirectory = '` + psQuote(workdir) + `'; $l.Arguments = '` + psQuote(arg) + `'; if (Test-Path '` + psQuote(icon) + `') { $l.IconLocation = '` + psQuote(icon) + `' }; $l.Description = '` + psQuote(appName+" · "+creditLine) + `'; $l.Save()`
	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-Command", ps)
	cmd.SysProcAttr = hideWindow()
	return cmd.Run()
}

func writeUninstallReg(dir, exe, uninst string) {
	if runtime.GOOS != "windows" {
		return
	}
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\` + uninstID
	vals := map[string]string{
		"DisplayName":     appName,
		"DisplayVersion":  appVersion,
		"Publisher":       publisher,
		"InstallLocation": dir,
		"DisplayIcon":     filepath.Join(dir, "app.ico"),
		"UninstallString": `"` + uninst + `" --uninstall`,
		"HelpLink":        "https://github.com/Johny-Silverhand/sklad-uchet",
		"Comments":        creditLine,
	}
	for k, v := range vals {
		cmd := exec.Command("reg", "add", key, "/v", k, "/t", "REG_SZ", "/d", v, "/f")
		cmd.SysProcAttr = hideWindow()
		_ = cmd.Run()
	}
	cmd := exec.Command("reg", "add", key, "/v", "NoModify", "/t", "REG_DWORD", "/d", "1", "/f")
	cmd.SysProcAttr = hideWindow()
	_ = cmd.Run()
}

func runUninstall() {
	dir := ""
	if exe, err := os.Executable(); err == nil {
		dir = filepath.Dir(exe)
	}
	if dir == "" {
		dir = defaultInstallDir()
	}
	_ = os.Remove(filepath.Join(os.Getenv("USERPROFILE"), "Desktop", appName+".lnk"))
	_ = os.RemoveAll(filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", publisher))
	_ = os.Remove(filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup", appName+".lnk"))
	cmd := exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\`+uninstID, "/f")
	cmd.SysProcAttr = hideWindow()
	_ = cmd.Run()
	_ = os.Remove(markerPath())
	bat := filepath.Join(os.TempDir(), "vlabs-sklad-unins.bat")
	body := "@echo off\r\nping 127.0.0.1 -n 2 >nul\r\nrd /s /q \"" + dir + "\"\r\ndel \"%~f0\"\r\n"
	_ = os.WriteFile(bat, []byte(body), 0o644)
	c := exec.Command("cmd", "/c", bat)
	c.SysProcAttr = hideWindow()
	_ = c.Start()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func licenseText() string {
	return appName + "\r\n" + creditLine + "\r\n© 2026 " + publisher + ".\r\n\r\n" +
		"Локальное Windows-приложение учёта склада (Go + SQLite + WebView2).\r\n" +
		"Запрещается копирование, декомпиляция и перепродажа без письменного согласия Victimok Labs.\r\n"
}

func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
