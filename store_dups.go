package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
)

func (s *Store) DuplicateGroups() ([]DuplicateGroup, error) {
	rows, err := s.db.Query(`
SELECT name_norm FROM items
GROUP BY name_norm
HAVING COUNT(*) > 1
ORDER BY name_norm`)
	if err != nil {
		return nil, err
	}
	var norms []string
	for rows.Next() {
		var norm string
		if err := rows.Scan(&norm); err != nil {
			_ = rows.Close()
			return nil, err
		}
		norms = append(norms, norm)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()

	var groups []DuplicateGroup
	for _, norm := range norms {
		items, err := s.FindByNorm(norm, -1)
		if err != nil {
			return nil, err
		}
		total := 0
		for _, it := range items {
			total += it.Quantity
		}
		groups = append(groups, DuplicateGroup{
			NormalizedName: norm,
			Items:          items,
			TotalQty:       total,
		})
	}
	if groups == nil {
		groups = []DuplicateGroup{}
	}
	return groups, nil
}

func pickBest(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if len(b) > len(a) {
		return b
	}
	return a
}

func (s *Store) Merge(primaryID int64, otherIDs []int64) (Item, error) {
	if primaryID <= 0 {
		return Item{}, fmt.Errorf("укажите основную запись")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Item{}, err
	}
	defer func() { _ = tx.Rollback() }()

	primary, err := getTx(tx, primaryID)
	if err != nil {
		return Item{}, err
	}

	seen := map[int64]bool{primaryID: true}
	var others []Item
	for _, id := range otherIDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		it, err := getTx(tx, id)
		if err != nil {
			return Item{}, fmt.Errorf("запись %d: %w", id, err)
		}
		others = append(others, it)
	}
	if len(others) == 0 {
		return Item{}, fmt.Errorf("нет записей для объединения")
	}

	qty := primary.Quantity
	sku := primary.SKU
	notes := primary.Notes
	cell := primary.Cell
	kind := primary.Kind
	extraCells := []string{}

	for _, o := range others {
		qty += o.Quantity
		sku = pickBest(sku, o.SKU)
		if strings.TrimSpace(o.Cell) != "" && !strings.EqualFold(strings.TrimSpace(o.Cell), strings.TrimSpace(cell)) {
			extraCells = append(extraCells, strings.TrimSpace(o.Cell))
		}
		if strings.TrimSpace(o.Notes) != "" {
			if notes == "" {
				notes = o.Notes
			} else if !strings.Contains(notes, o.Notes) {
				notes = notes + "\n" + o.Notes
			}
		}
		if kind == "" {
			kind = o.Kind
		}
	}

	if len(extraCells) > 0 {
		uniq := []string{}
		seenCell := map[string]bool{}
		for _, c := range extraCells {
			k := strings.ToUpper(c)
			if seenCell[k] {
				continue
			}
			seenCell[k] = true
			uniq = append(uniq, c)
		}
		line := "Также: " + strings.Join(uniq, ", ")
		if notes == "" {
			notes = line
		} else if !strings.Contains(notes, line) {
			notes = notes + "\n" + line
		}
	}

	ts := nowISO()
	_, err = tx.Exec(`UPDATE items SET quantity=?, sku=?, notes=?, cell=?, kind=?, updated_at=? WHERE id=?`,
		qty, sku, notes, cell, kind, ts, primaryID)
	if err != nil {
		return Item{}, err
	}
	for _, o := range others {
		if _, err := tx.Exec(`DELETE FROM items WHERE id=?`, o.ID); err != nil {
			return Item{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Item{}, err
	}
	return s.Get(primaryID)
}

func getTx(tx *sql.Tx, id int64) (Item, error) {
	row := tx.QueryRow(`SELECT `+itemCols+` FROM items WHERE id = ?`, id)
	return scanItem(row)
}

func (s *Store) Stats() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT kind, COUNT(*), COALESCE(SUM(quantity),0) FROM items GROUP BY kind`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{
		"total_items": 0,
		"total_qty":   0,
		"zapchast":    0,
		"ustroystvo":  0,
		"komplektuyushchee": 0,
	}
	for rows.Next() {
		var kind string
		var cnt, qty int
		if err := rows.Scan(&kind, &cnt, &qty); err != nil {
			return nil, err
		}
		out[kind] = cnt
		out["total_items"] += cnt
		out["total_qty"] += qty
	}
	return out, rows.Err()
}

func (s *Store) DBPath() string {
	dir, _ := dataDir()
	return filepath.Join(dir, "sklad.db")
}
