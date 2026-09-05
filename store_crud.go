package main

import (
	"fmt"
	"strings"
)

func (s *Store) List(kind, qName, qCell string) ([]Item, error) {
	where := []string{"1=1"}
	args := []any{}
	if kind != "" && kind != "all" {
		where = append(where, "kind = ?")
		args = append(args, kind)
	}
	if n := strings.TrimSpace(qName); n != "" {
		where = append(where, "name_norm LIKE ?")
		args = append(args, "%"+NormalizeName(n)+"%")
	}
	if c := strings.TrimSpace(qCell); c != "" {
		where = append(where, "UPPER(cell) LIKE ?")
		args = append(args, strings.ToUpper(c)+"%")
	}
	query := fmt.Sprintf(`SELECT %s FROM items WHERE %s ORDER BY name_norm ASC, id ASC`, itemCols, strings.Join(where, " AND "))
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if out == nil {
		out = []Item{}
	}
	return out, rows.Err()
}

func (s *Store) Get(id int64) (Item, error) {
	row := s.db.QueryRow(`SELECT `+itemCols+` FROM items WHERE id = ?`, id)
	return scanItem(row)
}

func (s *Store) FindByNorm(norm string, excludeID int64) ([]Item, error) {
	rows, err := s.db.Query(`SELECT `+itemCols+` FROM items WHERE name_norm = ? AND id != ? ORDER BY id`, norm, excludeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if out == nil {
		out = []Item{}
	}
	return out, rows.Err()
}

type UpsertInput struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Quantity  int    `json:"quantity"`
	Cell      string `json:"cell"`
	SKU       string `json:"sku"`
	Notes     string `json:"notes"`
	Force     bool   `json:"force"`
}

func (s *Store) Create(in UpsertInput) (Item, []Item, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Item{}, nil, fmt.Errorf("укажите название")
	}
	if !validKind(in.Kind) {
		return Item{}, nil, fmt.Errorf("неверный тип")
	}
	if in.Quantity < 0 {
		return Item{}, nil, fmt.Errorf("количество не может быть отрицательным")
	}
	norm := NormalizeName(name)
	dups, err := s.FindByNorm(norm, 0)
	if err != nil {
		return Item{}, nil, err
	}
	if len(dups) > 0 && !in.Force {
		return Item{}, dups, fmt.Errorf("duplicate")
	}
	ts := nowISO()
	res, err := s.db.Exec(`
INSERT INTO items (name, name_norm, kind, quantity, cell, sku, notes, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		name, norm, in.Kind, in.Quantity, strings.TrimSpace(in.Cell),
		strings.TrimSpace(in.SKU), strings.TrimSpace(in.Notes), ts, ts)
	if err != nil {
		return Item{}, nil, err
	}
	id, _ := res.LastInsertId()
	it, err := s.Get(id)
	return it, dups, err
}

func (s *Store) Update(id int64, in UpsertInput) (Item, []Item, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Item{}, nil, fmt.Errorf("укажите название")
	}
	if !validKind(in.Kind) {
		return Item{}, nil, fmt.Errorf("неверный тип")
	}
	if in.Quantity < 0 {
		return Item{}, nil, fmt.Errorf("количество не может быть отрицательным")
	}
	norm := NormalizeName(name)
	dups, err := s.FindByNorm(norm, id)
	if err != nil {
		return Item{}, nil, err
	}
	if len(dups) > 0 && !in.Force {
		return Item{}, dups, fmt.Errorf("duplicate")
	}
	ts := nowISO()
	res, err := s.db.Exec(`
UPDATE items SET name=?, name_norm=?, kind=?, quantity=?, cell=?, sku=?, notes=?, updated_at=?
WHERE id=?`,
		name, norm, in.Kind, in.Quantity, strings.TrimSpace(in.Cell),
		strings.TrimSpace(in.SKU), strings.TrimSpace(in.Notes), ts, id)
	if err != nil {
		return Item{}, nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Item{}, nil, fmt.Errorf("запись не найдена")
	}
	it, err := s.Get(id)
	return it, dups, err
}

func (s *Store) Delete(id int64) error {
	res, err := s.db.Exec(`DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("запись не найдена")
	}
	return nil
}

