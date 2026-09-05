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
	appName     = "Точка Склада"
	appExeName  = "TochkaSklada"
	appVersion  = "1.3.0"
	publisher   = "Victimok Labs"
	uninstID    = "VictimokLabsSkladUchet"
	creditLine  = "Разработано в Victimok Labs"
	defaultPort = "17890"
)

func main() {
	runtime.LockOSThread()
	defer func() {
		if rec := recover(); rec != nil {
			msg := fmt.Sprintf("Сбой запуска: %v\n\nПодробности: %%APPDATA%%\\VictimokLabs\\SkladUchet\\launch.log", rec)
			logLaunch("panic: %v", rec)
			showError(appName, msg)
			os.Exit(1)
		}
	}()

	mode := detectMode(os.Args[1:])
	argsForFlags := filterModeFlags(os.Args)
	flagSet := flag.NewFlagSet(appExeName, flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)
	noBrowser := flagSet.Bool("no-browser", false, "не открывать окно (отладка)")
	addrFlag := flagSet.String("addr", "", "локальный адрес UI (по умолчанию 127.0.0.1:"+defaultPort+")")
	_ = flagSet.Parse(argsForFlags[1:])

	addr := *addrFlag
	if addr == "" {
		addr = "127.0.0.1:" + defaultPort
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			fatal("Не удалось открыть локальный порт: " + err.Error())
			return
		}
	}

	mux := http.NewServeMux()
	attachInstallerAPI(mux, mode)

	var store *Store
	if mode == "app" {
		store, err = OpenStore()
		if err != nil {
			logLaunch("OpenStore: %v", err)
			fatal("Не удалось открыть базу данных:\n" + err.Error() + "\n\nПуть: %APPDATA%\\VictimokLabs\\SkladUchet\\sklad.db")
			return
		}
		defer store.Close()
		(&api{store: store}).mount(mux)
		logLaunch("mode=app db=%s", store.DBPath())
	} else {
		logLaunch("mode=%s", mode)
	}

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		fatal(err.Error())
		return
	}
	mux.Handle("/", spaHandler(sub, mode))

	srv := &http.Server{Handler: withLocalHandler(mux)}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	base := "http://" + ln.Addr().String()
	startURL := base + "/"
	switch mode {
	case "setup":
		startURL = base + "/setup/"
	case "uninstall":
		startURL = base + "/setup/uninstall.html"
	}

	waitReady(startURL)

	fmt.Println(appName, appVersion, "·", publisher)
	fmt.Println("mode:", mode)
	if store != nil {
		fmt.Println("DB:", store.DBPath())
	}
	fmt.Println(creditLine)

	if !*noBrowser {
		if runtime.GOOS == "windows" {
			logLaunch("webview url=%s", startURL)
			used := runNativeWindow(startURL, mode)
			if used {
				_ = srv.Close()
				return
			}
			logLaunch("webview unavailable, trying Edge/Chrome --app")
			showInfo(appName, "WebView2 недоступен. Пробую открыть через Edge/Chrome.\n\nЛучше установить WebView2 Runtime от Microsoft.")
		}
		cmd, err := openAppWindow(startURL, mode)
		if err != nil {
			logLaunch("openAppWindow: %v", err)
			fatal("Не удалось открыть окно приложения:\n" + err.Error())
			return
		}
		if cmd != nil && cmd.Process != nil {
			go func() {
				_, _ = cmd.Process.Wait()
				_ = srv.Close()
				os.Exit(0)
			}()
		} else {
			// rundll32 / xdg-open: keep process alive briefly so localhost UI serves
			time.Sleep(2 * time.Second)
		}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	_ = srv.Close()
	time.Sleep(100 * time.Millisecond)
}

func detectMode(args []string) string {
	mode := "setup"
	hasFlag := false
	for _, a := range args {
		switch a {
		case "--app", "-app":
			mode = "app"
			hasFlag = true
		case "--setup", "-setup":
			mode = "setup"
			hasFlag = true
		case "--uninstall", "-uninstall":
			mode = "uninstall"
			hasFlag = true
		}
	}
	if mode == "setup" {
		if exe, err := os.Executable(); err == nil {
			if _, err := os.Stat(filepath.Join(filepath.Dir(exe), "installed.json")); err == nil {
				mode = "app"
			}
		}
	}
	if !hasFlag && mode == "setup" {
		if _, err := os.Stat("go.mod"); err == nil {
			mode = "app"
		}
	}
	return mode
}

func filterModeFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for i, a := range args {
		if i == 0 {
			out = append(out, a)
			continue
		}
		switch a {
		case "--app", "-app", "--setup", "-setup", "--uninstall", "-uninstall":
			continue
		default:
			out = append(out, a)
		}
	}
	return out
}

func waitReady(url string) {
	client := &http.Client{Timeout: 200 * time.Millisecond, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode > 0 && resp.StatusCode < 500 {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func hasAppUI() bool {
	_, err := webFS.Open("web/app/index.html")
	return err == nil
}

func writeHTML(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}

func writeUIError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, `<!doctype html><html><body style="margin:0;background:#0b0f14;color:#e8eef7;font-family:Segoe UI,sans-serif;display:grid;place-items:center;height:100vh"><div style="text-align:center;padding:24px"><div style="font-size:18px;font-weight:600;margin-bottom:8px">%s</div><div style="opacity:.75">%s</div></div></body></html>`, appName, msg)
}

func serveEmbedHTML(w http.ResponseWriter, root fs.FS, name string) {
	b, err := fs.ReadFile(root, name)
	if err != nil {
		writeUIError(w, 404, "Не удалось загрузить интерфейс. Переустановите программу.")
		return
	}
	writeHTML(w, b)
}

func spaHandler(root fs.FS, mode string) http.Handler {
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/", "/index.html":
			switch mode {
			case "setup":
				serveEmbedHTML(w, root, "setup/index.html")
			case "uninstall":
				serveEmbedHTML(w, root, "setup/uninstall.html")
			default:
				if !hasAppUI() {
					serveEmbedHTML(w, root, "setup/index.html")
					return
				}
				serveEmbedHTML(w, root, "app/index.html")
			}
			return
		case "/setup", "/setup/", "/setup/index.html":
			serveEmbedHTML(w, root, "setup/index.html")
			return
		case "/setup/uninstall.html":
			serveEmbedHTML(w, root, "setup/uninstall.html")
			return
		}

		p := strings.TrimPrefix(path, "/")
		// Avoid FileServer's redirect from */index.html → ./ (breaks WebView).
		if strings.HasSuffix(p, "/index.html") || p == "index.html" {
			serveEmbedHTML(w, root, p)
			return
		}
		if p != "" {
			if st, err := fs.Stat(root, p); err == nil && !st.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}
		if hasAppUI() && !strings.HasPrefix(path, "/setup") && !strings.HasPrefix(path, "/api") {
			serveEmbedHTML(w, root, "app/index.html")
			return
		}
		writeUIError(w, 404, "Интерфейс не найден. Переустановите программу.")
	})
}

func withLocalHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	logLaunch("fatal: %s", msg)
	showError(appName, msg)
	os.Exit(1)
}

func logLaunch(format string, args ...any) {
	dir, err := dataDir()
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "launch.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}
