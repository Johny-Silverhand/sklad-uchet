package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type api struct {
	store *Store
}

func (a *api) mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", a.health)
	mux.HandleFunc("/api/stats", a.stats)
	mux.HandleFunc("/api/items", a.items)
	mux.HandleFunc("/api/items/", a.itemByID)
	mux.HandleFunc("/api/themes", a.themes)
	mux.HandleFunc("/api/themes/", a.themeByID)
	mux.HandleFunc("/api/duplicates", a.duplicates)
	mux.HandleFunc("/api/duplicates/merge", a.merge)
	mux.HandleFunc("/api/export.csv", a.exportCSV)
	mux.HandleFunc("/api/import.csv", a.importCSV)
	mux.HandleFunc("/api/backup", a.backup)
	mux.HandleFunc("/api/restore", a.restore)
	mux.HandleFunc("/api/movements", a.movements)
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
		"ok":        true,
		"app":       appName,
		"version":   appVersion,
		"db":        a.store.DBPath(),
		"credit":    creditLine,
		"publisher": publisher,
		"shell":     "webview2",
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
		list, err := a.store.List(f)
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

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (a *api) itemByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/items/")
	rest = strings.Trim(rest, "/")
	parts := strings.Split(rest, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, 400, map[string]string{"error": "Неверный id"})
		return
	}
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if action == "adjust" && r.Method == http.MethodPost {
		var body struct {
			Delta int `json:"delta"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Некорректный JSON"})
			return
		}
		it, err := a.store.Adjust(id, body.Delta)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"item": it})
		return
	}
	if action == "move" && r.Method == http.MethodPost {
		var body struct {
			Cell string `json:"cell"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Некорректный JSON"})
			return
		}
		it, err := a.store.MoveToCell(id, body.Cell)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"item": it})
		return
	}
	if action == "storage" && r.Method == http.MethodPost {
		var body struct {
			Storage string `json:"storage"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Некорректный JSON"})
			return
		}
		it, err := a.store.SetStorage(id, body.Storage)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"item": it})
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

func (a *api) themes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := a.store.ListThemes()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"themes": list})
	case http.MethodPost:
		var body struct {
			Name      string `json:"name"`
			SortOrder int    `json:"sort_order"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Некорректный JSON"})
			return
		}
		th, err := a.store.CreateTheme(body.Name, body.SortOrder)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 201, map[string]any{"theme": th})
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (a *api) themeByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/themes/")
	rest = strings.Trim(rest, "/")
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, 400, map[string]string{"error": "Неверный id"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		th, err := a.store.GetTheme(id)
		if err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"theme": th})
	case http.MethodPut, http.MethodPatch:
		var body struct {
			Name      string `json:"name"`
			SortOrder int    `json:"sort_order"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Некорректный JSON"})
			return
		}
		th, err := a.store.UpdateTheme(id, body.Name, body.SortOrder)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"theme": th})
	case http.MethodDelete:
		if err := a.store.DeleteTheme(id); err != nil {
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

func (a *api) exportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	name := "sklad-uchet-" + time.Now().Format("20060102") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	if err := a.store.ExportCSV(w); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
	}
}

func (a *api) importCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	update := r.URL.Query().Get("update") != "0"
	var reader io.Reader
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(16 << 20); err != nil {
			writeJSON(w, 400, map[string]string{"error": "Не удалось прочитать файл"})
			return
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": "Нужен файл file"})
			return
		}
		defer f.Close()
		reader = f
	} else {
		reader = io.LimitReader(r.Body, 16<<20)
	}
	res, err := a.store.ImportCSV(reader, update)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, res)
}

func (a *api) backup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	_ = readJSON(r, &body)
	path, err := a.store.BackupTo(body.Path)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "path": path})
}

func (a *api) restore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := readJSON(r, &body); err != nil || strings.TrimSpace(body.Path) == "" {
		writeJSON(w, 400, map[string]string{"error": "Укажите path к .db"})
		return
	}
	if err := a.store.RestoreFrom(body.Path); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (a *api) movements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := a.store.RecentMovements(limit)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"movements": list})
}
