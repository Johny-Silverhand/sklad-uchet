package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (s *Store) addMovement(itemID int64, kind string, delta int, fromCell, toCell, note string) error {
	_, err := s.db.Exec(`
INSERT INTO movements (item_id, kind, delta, from_cell, to_cell, note, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		itemID, kind, delta, fromCell, toCell, note, nowISO())
	return err
}

func (s *Store) Adjust(id int64, delta int) (Item, error) {
	if delta == 0 {
		return s.Get(id)
	}
	old, err := s.Get(id)
	if err != nil {
		return Item{}, fmt.Errorf("запись не найдена")
	}
	newQty := old.Quantity + delta
	if newQty < 0 {
		return Item{}, fmt.Errorf("количество не может быть отрицательным")
	}
	ts := nowISO()
	_, err = s.db.Exec(`UPDATE items SET quantity=?, updated_at=? WHERE id=?`, newQty, ts, id)
	if err != nil {
		return Item{}, err
	}
	_ = s.addMovement(id, "adjust", delta, old.Cell, old.Cell, fmt.Sprintf("%+d", delta))
	return s.Get(id)
}

func (s *Store) MoveToCell(id int64, cell string) (Item, error) {
	cell = strings.TrimSpace(cell)
	old, err := s.Get(id)
	if err != nil {
		return Item{}, fmt.Errorf("запись не найдена")
	}
	if strings.EqualFold(old.Cell, cell) {
		return old, nil
	}
	ts := nowISO()
	_, err = s.db.Exec(`UPDATE items SET cell=?, updated_at=? WHERE id=?`, cell, ts, id)
	if err != nil {
		return Item{}, err
	}
	_ = s.addMovement(id, "move", 0, old.Cell, cell, "перемещение")
	return s.Get(id)
}

func (s *Store) RecentMovements(limit int) ([]Movement, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.db.Query(`
SELECT id, item_id, kind, delta, from_cell, to_cell, note, created_at
FROM movements ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Movement
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.ItemID, &m.Kind, &m.Delta, &m.FromCell, &m.ToCell, &m.Note, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if out == nil {
		out = []Movement{}
	}
	return out, rows.Err()
}

func (s *Store) ExportCSV(w io.Writer) error {
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	if err := cw.Write([]string{"id", "name", "kind", "quantity", "min_qty", "cell", "sku", "notes"}); err != nil {
		return err
	}
	items, err := s.List(ListFilter{})
	if err != nil {
		return err
	}
	for _, it := range items {
		if err := cw.Write([]string{
			strconv.FormatInt(it.ID, 10),
			it.Name,
			it.Kind,
			strconv.Itoa(it.Quantity),
			strconv.Itoa(it.MinQty),
			it.Cell,
			it.SKU,
			it.Notes,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

type ImportResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

func (s *Store) ImportCSV(r io.Reader, updateExisting bool) (ImportResult, error) {
	data, err := io.ReadAll(io.LimitReader(r, 16<<20))
	if err != nil {
		return ImportResult{}, err
	}
	data = []byte(strings.TrimPrefix(string(data), "\ufeff"))

	parse := func(comma rune) ([][]string, error) {
		cr := csv.NewReader(strings.NewReader(string(data)))
		cr.Comma = comma
		cr.FieldsPerRecord = -1
		cr.LazyQuotes = true
		return cr.ReadAll()
	}
	rows, err := parse(';')
	if err != nil || (len(rows) > 0 && len(rows[0]) < 2) {
		rows2, err2 := parse(',')
		if err2 == nil {
			rows, err = rows2, nil
		}
	}
	if err != nil {
		return ImportResult{}, fmt.Errorf("не удалось разобрать CSV: %w", err)
	}

	res := ImportResult{Errors: []string{}}
	if len(rows) == 0 {
		return res, nil
	}
	start := 0
	header := map[string]int{}
	first := rows[0]
	looksHeader := false
	for i, h := range first {
		key := strings.ToLower(strings.TrimSpace(h))
		header[key] = i
		if key == "name" || key == "название" || key == "kind" || key == "тип" {
			looksHeader = true
		}
	}
	if looksHeader {
		start = 1
	} else {
		header = map[string]int{
			"name": 0, "kind": 1, "quantity": 2, "min_qty": 3, "cell": 4, "sku": 5, "notes": 6,
		}
	}
	col := func(row []string, keys ...string) string {
		for _, k := range keys {
			if i, ok := header[k]; ok && i < len(row) {
				return strings.TrimSpace(row[i])
			}
		}
		return ""
	}
	for i := start; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			res.Skipped++
			continue
		}
		name := col(row, "name", "название")
		kind := col(row, "kind", "тип")
		switch strings.ToLower(kind) {
		case "запчасть", "zapchast":
			kind = KindZapchast
		case "устройство", "ustroystvo":
			kind = KindUstroystvo
		case "комплектующее", "komplektuyushchee":
			kind = KindKomplektuyushchee
		}
		qty, _ := strconv.Atoi(col(row, "quantity", "кол-во", "qty"))
		minQty, _ := strconv.Atoi(col(row, "min_qty", "мин", "min"))
		cell := col(row, "cell", "ячейка")
		sku := col(row, "sku", "артикул")
		notes := col(row, "notes", "заметки")
		idStr := col(row, "id")
		if name == "" {
			res.Skipped++
			res.Errors = append(res.Errors, fmt.Sprintf("строка %d: нет названия", i+1))
			continue
		}
		if !validKind(kind) {
			kind = KindZapchast
		}
		if idStr != "" && updateExisting {
			id, _ := strconv.ParseInt(idStr, 10, 64)
			if id > 0 {
				if _, err := s.Get(id); err == nil {
					_, _, err := s.Update(id, UpsertInput{
						Name: name, Kind: kind, Quantity: qty, MinQty: minQty,
						Cell: cell, SKU: sku, Notes: notes, Force: true,
					})
					if err != nil {
						res.Errors = append(res.Errors, fmt.Sprintf("строка %d: %v", i+1, err))
						res.Skipped++
						continue
					}
					res.Updated++
					continue
				}
			}
		}
		norm := NormalizeName(name)
		dups, _ := s.FindByNorm(norm, 0)
		matched := false
		if updateExisting {
			for _, d := range dups {
				if strings.EqualFold(d.Cell, cell) || cell == "" {
					_, _, err := s.Update(d.ID, UpsertInput{
						Name: name, Kind: kind, Quantity: qty, MinQty: minQty,
						Cell: cell, SKU: sku, Notes: notes, Force: true,
					})
					if err != nil {
						res.Errors = append(res.Errors, fmt.Sprintf("строка %d: %v", i+1, err))
						res.Skipped++
					} else {
						res.Updated++
					}
					matched = true
					break
				}
			}
		}
		if matched {
			continue
		}
		_, _, err := s.Create(UpsertInput{
			Name: name, Kind: kind, Quantity: qty, MinQty: minQty,
			Cell: cell, SKU: sku, Notes: notes, Force: true,
		})
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("строка %d: %v", i+1, err))
			res.Skipped++
			continue
		}
		res.Created++
	}
	if len(res.Errors) > 20 {
		res.Errors = res.Errors[:20]
	}
	return res, nil
}

func (s *Store) BackupTo(destPath string) (string, error) {
	if strings.TrimSpace(destPath) == "" {
		dir, err := dataDir()
		if err != nil {
			return "", err
		}
		backups := filepath.Join(dir, "backups")
		_ = os.MkdirAll(backups, 0o755)
		destPath = filepath.Join(backups, "sklad-"+time.Now().Format("20060102-150405")+".db")
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", err
	}
	_, _ = s.db.Exec(`PRAGMA wal_checkpoint(FULL)`)
	if err := copyFileSimple(s.dbPath, destPath); err != nil {
		return "", err
	}
	return destPath, nil
}

func (s *Store) RestoreFrom(srcPath string) error {
	srcPath = strings.TrimSpace(srcPath)
	if srcPath == "" {
		return fmt.Errorf("укажите путь к файлу резервной копии")
	}
	st, err := os.Stat(srcPath)
	if err != nil || st.IsDir() {
		return fmt.Errorf("файл резервной копии не найден")
	}
	_, _ = s.db.Exec(`PRAGMA wal_checkpoint(FULL)`)
	_ = s.db.Close()
	bak := s.dbPath + ".before-restore"
	_ = copyFileSimple(s.dbPath, bak)
	if err := copyFileSimple(srcPath, s.dbPath); err != nil {
		_ = copyFileSimple(bak, s.dbPath)
		db, e2 := reopenDB(s.dbPath)
		if e2 == nil {
			s.db = db
		}
		return err
	}
	_ = os.Remove(s.dbPath + "-wal")
	_ = os.Remove(s.dbPath + "-shm")
	db, err := reopenDB(s.dbPath)
	if err != nil {
		return err
	}
	s.db = db
	return s.migrate()
}

func reopenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func copyFileSimple(src, dst string) error {
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
