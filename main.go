package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

//go:embed all:web
var webFS embed.FS

const (
	appName    = "Склад Учёт"
	appVersion = "1.0.0"
	publisher  = "Victimok Labs"
	creditLine = "Разработано в Victimok Labs"
)

func main() {
	noBrowser := flag.Bool("no-browser", false, "не открывать окно браузера")
	addrFlag := flag.String("addr", "127.0.0.1:0", "адрес HTTP-сервера")
	flag.Parse()

	store, err := OpenStore()
	if err != nil {
		fatal("База данных: " + err.Error())
		return
	}
	defer store.Close()

	ln, err := net.Listen("tcp", *addrFlag)
	if err != nil {
		fatal("Не удалось открыть порт: " + err.Error())
		return
	}

	mux := http.NewServeMux()
	(&api{store: store}).mount(mux)

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		fatal(err.Error())
		return
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	srv := &http.Server{Handler: withCORS(mux)}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	url := "http://" + ln.Addr().String() + "/"
	fmt.Println(appName, appVersion)
	fmt.Println("UI:", url)
	fmt.Println("DB:", store.DBPath())

	var browserCmd *exec.Cmd
	if !*noBrowser {
		cmd, err := openAppWindow(url)
		if err != nil {
			fmt.Fprintln(os.Stderr, "браузер:", err)
		} else if cmd != nil && cmd.Process != nil {
			browserCmd = cmd
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
	if browserCmd != nil && browserCmd.Process != nil {
		_ = browserCmd.Process.Kill()
	}
	time.Sleep(100 * time.Millisecond)
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
