package main

import (
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func (a *api) serviceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/report.html", a.reportHTML)
	mux.HandleFunc("/api/db/integrity", a.dbIntegrity)
	mux.HandleFunc("/api/db/vacuum", a.dbVacuum)
	mux.HandleFunc("/api/paths", a.paths)
	mux.HandleFunc("/api/reveal-data", a.revealData)
	mux.HandleFunc("/api/cells/crowded", a.crowdedCells)
}

func (a *api) listFilterFromQuery(r *http.Request) ListFilter {
	low := r.URL.Query().Get("low") == "1" || r.URL.Query().Get("low_stock") == "1"
	f := ListFilter{
		Kind:     r.URL.Query().Get("kind"),
		QName:    firstNonEmpty(r.URL.Query().Get("q"), r.URL.Query().Get("name")),
		QCell:    r.URL.Query().Get("cell"),
		LowStock: low,
		Storage:  r.URL.Query().Get("storage"),
	}
	themeQ := r.URL.Query().Get("theme_id")
	if themeQ == "none" || themeQ == "0" {
		f.NoTheme = true
	} else if themeQ != "" {
		if id, err := strconv.ParseInt(themeQ, 10, 64); err == nil && id > 0 {
			f.ThemeID = &id
			f.ThemeOnly = true
		}
	}
	return f
}

func (a *api) reportHTML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	f := a.listFilterFromQuery(r)
	list, err := a.store.List(f)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	parts := []string{}
	if f.Storage == StorageBalance {
		parts = append(parts, "хранение: Баланс")
	} else if f.Storage == StorageTemporary {
		parts = append(parts, "хранение: Временное")
	} else {
		parts = append(parts, "хранение: всё")
	}
	if f.NoTheme {
		parts = append(parts, "тема: Без темы")
	} else if f.ThemeOnly && f.ThemeID != nil {
		name := "id=" + strconv.FormatInt(*f.ThemeID, 10)
		if th, err := a.store.GetTheme(*f.ThemeID); err == nil {
			name = th.Name
		}
		parts = append(parts, "тема: "+name)
	} else {
		parts = append(parts, "тема: Все")
	}
	if f.Kind != "" && f.Kind != "all" {
		parts = append(parts, "тип: "+kindLabelRU(f.Kind))
	}
	if f.LowStock {
		parts = append(parts, "только мало")
	}
	filterDesc := strings.Join(parts, "; ")
	now := time.Now().UTC()
	iso := now.Format(time.RFC3339)
	// Moscow-friendly stamp (UTC+3); client JS may overwrite with local locale string
	msk := now.Add(3 * time.Hour).Format("02.01.2006 15:04") + " (МСК)"
	html := buildReportHTML(list, filterDesc, iso, msk)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(html))
}

func (a *api) dbIntegrity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	res, err := a.store.IntegrityCheck()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": strings.EqualFold(res, "ok"), "result": res})
}

func (a *api) dbVacuum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if err := a.store.Vacuum(); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *api) paths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, 200, map[string]any{
		"db":       a.store.DBPath(),
		"data_dir": a.store.DataDir(),
		"os":       runtime.GOOS,
	})
}

func (a *api) revealData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if err := revealPath(a.store.DataDir()); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "path": a.store.DataDir()})
}

func (a *api) crowdedCells(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	min := 1
	if v := r.URL.Query().Get("min"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			min = n
		}
	}
	list, err := a.store.CrowdedCells(min)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"cells": list, "min": min})
}
