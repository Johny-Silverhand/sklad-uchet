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
	minQty := primary.MinQty
	sku := primary.SKU
	notes := primary.Notes
	cell := primary.Cell
	kind := primary.Kind
	extraCells := []string{}

	for _, o := range others {
		qty += o.Quantity
		if o.MinQty > minQty {
			minQty = o.MinQty
		}
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
	_, err = tx.Exec(`UPDATE items SET quantity=?, min_qty=?, sku=?, notes=?, cell=?, kind=?, updated_at=? WHERE id=?`,
		qty, minQty, sku, notes, cell, kind, ts, primaryID)
	if err != nil {
		return Item{}, err
	}
	for _, o := range others {
		if _, err := tx.Exec(`DELETE FROM items WHERE id=?`, o.ID); err != nil {
			return Item{}, err
		}
	}
	if _, err := tx.Exec(`
INSERT INTO movements (item_id, kind, delta, from_cell, to_cell, note, created_at)
VALUES (?, 'merge', ?, ?, ?, ?, ?)`,
		primaryID, qty-primary.Quantity, primary.Cell, cell,
		fmt.Sprintf("объединение %d записей", len(others)+1), ts); err != nil {
		return Item{}, err
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

func (s *Store) Stats() (Stats, error) {
	out := Stats{
		ByKind: map[string]int{
			KindZapchast: 0, KindUstroystvo: 0, KindKomplektuyushchee: 0,
		},
		QtyByKind: map[string]int{
			KindZapchast: 0, KindUstroystvo: 0, KindKomplektuyushchee: 0,
		},
		TopCells: []CellStat{},
	}
	rows, err := s.db.Query(`SELECT kind, COUNT(*), COALESCE(SUM(quantity),0) FROM items GROUP BY kind`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var kind string
		var cnt, qty int
		if err := rows.Scan(&kind, &cnt, &qty); err != nil {
			_ = rows.Close()
			return out, err
		}
		out.ByKind[kind] = cnt
		out.QtyByKind[kind] = qty
		out.TotalItems += cnt
		out.TotalQty += qty
	}
	_ = rows.Close()

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM items WHERE min_qty > 0 AND quantity <= min_qty`).Scan(&out.LowStock); err != nil {
		return out, err
	}

	crows, err := s.db.Query(`
SELECT cell, COUNT(*), COALESCE(SUM(quantity),0)
FROM items WHERE TRIM(cell) != ''
GROUP BY cell ORDER BY COUNT(*) DESC, SUM(quantity) DESC LIMIT 8`)
	if err != nil {
		return out, err
	}
	defer crows.Close()
	for crows.Next() {
		var cs CellStat
		if err := crows.Scan(&cs.Cell, &cs.Count, &cs.Qty); err != nil {
			return out, err
		}
		out.TopCells = append(out.TopCells, cs)
	}
	return out, crows.Err()
}

func (s *Store) DBPath() string {
	if s.dbPath != "" {
		return s.dbPath
	}
	dir, _ := dataDir()
	return filepath.Join(dir, "sklad.db")
}
