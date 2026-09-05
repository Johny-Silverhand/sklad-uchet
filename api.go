package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type api struct {
	store *Store
}

func (a *api) mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", a.health)
	mux.HandleFunc("/api/stats", a.stats)
	mux.HandleFunc("/api/items", a.items)
	mux.HandleFunc("/api/items/", a.itemByID)
	mux.HandleFunc("/api/duplicates", a.duplicates)
	mux.HandleFunc("/api/duplicates/merge", a.merge)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	return dec.Decode(dst)
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"ok":     true,
		"app":    appName,
		"db":     a.store.DBPath(),
		"credit": creditLine,
	})
}

func (a *api) stats(w http.ResponseWriter, r *http.Request) {
	st, err := a.store.Stats()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, st)
}

func (a *api) items(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		kind := r.URL.Query().Get("kind")
		qName := r.URL.Query().Get("q")
		if qName == "" {
			qName = r.URL.Query().Get("name")
		}
		qCell := r.URL.Query().Get("cell")
		list, err := a.store.List(kind, qName, qCell)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"items": list})
	case http.MethodPost:
		var in UpsertInput
		if err := readJSON(r, &in); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Некорректный JSON"})
			return
		}
		it, dups, err := a.store.Create(in)
		if err != nil {
			if err.Error() == "duplicate" {
				writeJSON(w, 409, map[string]any{
					"error":      "Найдены записи с похожим названием",
					"duplicates": dups,
				})
				return
			}
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 201, map[string]any{"item": it, "warned_duplicates": dups})
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (a *api) itemByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/items/")
	idStr = strings.Trim(idStr, "/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, 400, map[string]string{"error": "Неверный id"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		it, err := a.store.Get(id)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": "Не найдено"})
			return
		}
		writeJSON(w, 200, map[string]any{"item": it})
	case http.MethodPut, http.MethodPatch:
		var in UpsertInput
		if err := readJSON(r, &in); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Некорректный JSON"})
			return
		}
		it, dups, err := a.store.Update(id, in)
		if err != nil {
			if err.Error() == "duplicate" {
				writeJSON(w, 409, map[string]any{
					"error":      "Найдены записи с похожим названием",
					"duplicates": dups,
				})
				return
			}
			code := 400
			if strings.Contains(err.Error(), "не найдена") {
				code = 404
			}
			writeJSON(w, code, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"item": it, "warned_duplicates": dups})
	case http.MethodDelete:
		if err := a.store.Delete(id); err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (a *api) duplicates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	groups, err := a.store.DuplicateGroups()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"groups": groups})
}

func (a *api) merge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		PrimaryID int64   `json:"primary_id"`
		OtherIDs  []int64 `json:"other_ids"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "Некорректный JSON"})
		return
	}
	it, err := a.store.Merge(body.PrimaryID, body.OtherIDs)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"item": it})
}
