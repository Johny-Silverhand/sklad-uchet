package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

//go:embed all:web
var webFS embed.FS

const (
	appName     = "Склад Учёт"
	appExeName  = "SkladUchet"
	appVersion  = "1.2.0"
	publisher   = "Victimok Labs"
	uninstID    = "VictimokLabsSkladUchet"
	creditLine  = "Разработано в Victimok Labs"
	defaultPort = "17890"
)

func main() {
	noBrowser := flag.Bool("no-browser", false, "не открывать окно (только HTTP на localhost)")
	addrFlag := flag.String("addr", "", "адрес HTTP-сервера (по умолчанию 127.0.0.1:"+defaultPort+")")
	flag.Parse()

	mode := detectMode()
	addr := *addrFlag
	if addr == "" {
		addr = "127.0.0.1:" + defaultPort
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			fatal("Не удалось открыть порт: " + err.Error())
			return
		}
	}

	mux := http.NewServeMux()
	attachInstallerAPI(mux, mode)

	var store *Store
	if mode == "app" {
		store, err = OpenStore()
		if err != nil {
			fatal("База данных: " + err.Error())
			return
		}
		defer store.Close()
		(&api{store: store}).mount(mux)
	}

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		fatal(err.Error())
		return
	}
	mux.Handle("/", spaHandler(sub, mode))

	srv := &http.Server{Handler: withCORS(mux)}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	base := "http://" + ln.Addr().String()
	startURL := base + "/"
	switch mode {
	case "setup":
		startURL = base + "/setup/index.html"
	case "uninstall":
		startURL = base + "/setup/uninstall.html"
	}

	fmt.Println(appName, appVersion, "·", publisher)
	fmt.Println("mode:", mode)
	fmt.Println("UI:", startURL)
	if store != nil {
		fmt.Println("DB:", store.DBPath())
	}
	fmt.Println(creditLine)

	if !*noBrowser {
		// Prefer native WebView2 window on Windows (real desktop app, no browser chrome).
		if runtime.GOOS == "windows" {
			used := runNativeWindow(startURL, mode)
			_ = srv.Close()
			if used {
				return
			}
			// Fallback only if WebView2 runtime missing.
			fmt.Fprintln(os.Stderr, "fallback: browser --app=")
		}
		cmd, err := openAppWindow(startURL, mode)
		if err != nil {
			fmt.Fprintln(os.Stderr, "окно:", err)
		} else if cmd != nil && cmd.Process != nil {
			go func() {
				_, _ = cmd.Process.Wait()
				_ = srv.Close()
				os.Exit(0)
			}()
		}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	_ = srv.Close()
	time.Sleep(100 * time.Millisecond)
}

func detectMode() string {
	mode := "setup"
	for _, a := range os.Args[1:] {
		switch a {
		case "--app", "-app":
			mode = "app"
		case "--setup", "-setup":
			mode = "setup"
		case "--uninstall", "-uninstall":
			mode = "uninstall"
		}
	}
	if mode == "setup" {
		if exe, err := os.Executable(); err == nil {
			if _, err := os.Stat(filepath.Join(filepath.Dir(exe), "installed.json")); err == nil {
				mode = "app"
			}
		}
	}
	hasFlag := false
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "--app") || strings.HasPrefix(a, "--setup") || strings.HasPrefix(a, "--uninstall") ||
			a == "-app" || a == "-setup" || a == "-uninstall" {
			hasFlag = true
			break
		}
	}
	if !hasFlag && mode == "setup" {
		if _, err := os.Stat("go.mod"); err == nil {
			mode = "app"
		}
	}
	return mode
}

func hasAppUI() bool {
	_, err := webFS.Open("web/app/index.html")
	return err == nil
}

func spaHandler(root fs.FS, mode string) http.Handler {
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && mode == "setup" {
			http.Redirect(w, r, "/setup/index.html", http.StatusFound)
			return
		}
		if r.URL.Path == "/" && mode == "uninstall" {
			http.Redirect(w, r, "/setup/uninstall.html", http.StatusFound)
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			if !hasAppUI() {
				http.Redirect(w, r, "/setup/index.html", http.StatusFound)
				return
			}
			b, err := fs.ReadFile(root, "app/index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(b)
			return
		}
		if _, err := fs.Stat(root, p); err == nil {
			files.ServeHTTP(w, r)
			return
		}
		if hasAppUI() && !strings.HasPrefix(r.URL.Path, "/setup") && !strings.HasPrefix(r.URL.Path, "/api") {
			b, err := fs.ReadFile(root, "app/index.html")
			if err == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(b)
				return
			}
		}
		http.NotFound(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
