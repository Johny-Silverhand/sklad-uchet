package main

import (
	"fmt"
	"strings"
)

type ListFilter struct {
	Kind      string
	QName     string
	QCell     string
	LowStock  bool
	Storage   string
	ThemeID   *int64
	ThemeOnly bool
	NoTheme   bool
}

func (s *Store) List(f ListFilter) ([]Item, error) {
	where := []string{"1=1"}
	args := []any{}
	if f.Kind != "" && f.Kind != "all" {
		where = append(where, "i.kind = ?")
		args = append(args, f.Kind)
	}
	if st := strings.TrimSpace(f.Storage); st != "" && st != "all" {
		where = append(where, "i.storage = ?")
		args = append(args, st)
	}
	if n := strings.TrimSpace(f.QName); n != "" {
		where = append(where, "i.name_norm LIKE ?")
		args = append(args, "%"+NormalizeName(n)+"%")
	}
	if c := strings.TrimSpace(f.QCell); c != "" {
		where = append(where, "UPPER(i.cell) LIKE ?")
		args = append(args, strings.ToUpper(c)+"%")
	}
	if f.LowStock {
		where = append(where, "i.min_qty > 0 AND i.quantity <= i.min_qty")
	}
	if f.NoTheme {
		where = append(where, "i.theme_id IS NULL")
	} else if f.ThemeOnly && f.ThemeID != nil {
		where = append(where, "i.theme_id = ?")
		args = append(args, *f.ThemeID)
	}
	query := fmt.Sprintf(
		`SELECT %s FROM %s WHERE %s ORDER BY COALESCE(t.sort_order, 999999), COALESCE(t.name, ''), i.name_norm ASC, i.id ASC`,
		itemCols, itemFrom, strings.Join(where, " AND "),
	)
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
	row := s.db.QueryRow(`SELECT `+itemCols+` FROM `+itemFrom+` WHERE i.id = ?`, id)
	return scanItem(row)
}

func (s *Store) FindByNorm(norm string, excludeID int64) ([]Item, error) {
	rows, err := s.db.Query(`SELECT `+itemCols+` FROM `+itemFrom+` WHERE i.name_norm = ? AND i.id != ? ORDER BY i.id`, norm, excludeID)
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
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Quantity int    `json:"quantity"`
	MinQty   int    `json:"min_qty"`
	Cell     string `json:"cell"`
	SKU      string `json:"sku"`
	Notes    string `json:"notes"`
	Storage  string `json:"storage"`
	ThemeID  *int64 `json:"theme_id"`
	Force    bool   `json:"force"`
}

func normalizeUpsert(in *UpsertInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return fmt.Errorf("укажите название")
	}
	if !validKind(in.Kind) {
		return fmt.Errorf("неверный тип")
	}
	if in.Quantity < 0 {
		return fmt.Errorf("количество не может быть отрицательным")
	}
	if in.MinQty < 0 {
		return fmt.Errorf("мин. остаток не может быть отрицательным")
	}
	if in.Storage == "" {
		in.Storage = StorageBalance
	}
	if !validStorage(in.Storage) {
		return fmt.Errorf("неверное хранение (balance/temporary)")
	}
	return nil
}

func (s *Store) resolveThemeID(themeID *int64) (any, error) {
	if themeID == nil {
		return nil, nil
	}
	id := *themeID
	if id <= 0 {
		return nil, nil
	}
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM themes WHERE id = ?`, id).Scan(&n)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("тема не найдена")
	}
	return id, nil
}

func (s *Store) Create(in UpsertInput) (Item, []Item, error) {
	if err := normalizeUpsert(&in); err != nil {
		return Item{}, nil, err
	}
	norm := NormalizeName(in.Name)
	dups, err := s.FindByNorm(norm, 0)
	if err != nil {
		return Item{}, nil, err
	}
	if len(dups) > 0 && !in.Force {
		return Item{}, dups, fmt.Errorf("duplicate")
	}
	themeVal, err := s.resolveThemeID(in.ThemeID)
	if err != nil {
		return Item{}, nil, err
	}
	ts := nowISO()
	res, err := s.db.Exec(`
INSERT INTO items (name, name_norm, kind, quantity, min_qty, cell, sku, notes, storage, theme_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.Name, norm, in.Kind, in.Quantity, in.MinQty, strings.TrimSpace(in.Cell),
		strings.TrimSpace(in.SKU), strings.TrimSpace(in.Notes), in.Storage, themeVal, ts, ts)
	if err != nil {
		return Item{}, nil, err
	}
	id, _ := res.LastInsertId()
	_ = s.addMovement(id, "create", in.Quantity, "", strings.TrimSpace(in.Cell), "создание")
	it, err := s.Get(id)
	return it, dups, err
}

func (s *Store) Update(id int64, in UpsertInput) (Item, []Item, error) {
	if err := normalizeUpsert(&in); err != nil {
		return Item{}, nil, err
	}
	old, err := s.Get(id)
	if err != nil {
		return Item{}, nil, fmt.Errorf("запись не найдена")
	}
	norm := NormalizeName(in.Name)
	dups, err := s.FindByNorm(norm, id)
	if err != nil {
		return Item{}, nil, err
	}
	if len(dups) > 0 && !in.Force {
		return Item{}, dups, fmt.Errorf("duplicate")
	}
	themeVal, err := s.resolveThemeID(in.ThemeID)
	if err != nil {
		return Item{}, nil, err
	}
	ts := nowISO()
	res, err := s.db.Exec(`
UPDATE items SET name=?, name_norm=?, kind=?, quantity=?, min_qty=?, cell=?, sku=?, notes=?, storage=?, theme_id=?, updated_at=?
WHERE id=?`,
		in.Name, norm, in.Kind, in.Quantity, in.MinQty, strings.TrimSpace(in.Cell),
		strings.TrimSpace(in.SKU), strings.TrimSpace(in.Notes), in.Storage, themeVal, ts, id)
	if err != nil {
		return Item{}, nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Item{}, nil, fmt.Errorf("запись не найдена")
	}
	delta := in.Quantity - old.Quantity
	if delta != 0 || strings.TrimSpace(in.Cell) != old.Cell || in.Storage != old.Storage {
		note := "редактирование"
		if in.Storage != old.Storage {
			note = "смена хранения → " + in.Storage
		}
		_ = s.addMovement(id, "edit", delta, old.Cell, strings.TrimSpace(in.Cell), note)
	}
	it, err := s.Get(id)
	return it, dups, err
}

func (s *Store) Delete(id int64) error {
	old, err := s.Get(id)
	if err != nil {
		return fmt.Errorf("запись не найдена")
	}
	res, err := s.db.Exec(`DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("запись не найдена")
	}
	_ = s.addMovement(id, "delete", -old.Quantity, old.Cell, "", "удаление «"+old.Name+"»")
	return nil
}

func (s *Store) SetStorage(id int64, storage string) (Item, error) {
	if !validStorage(storage) {
		return Item{}, fmt.Errorf("неверное хранение")
	}
	old, err := s.Get(id)
	if err != nil {
		return Item{}, fmt.Errorf("запись не найдена")
	}
	if old.Storage == storage {
		return old, nil
	}
	ts := nowISO()
	_, err = s.db.Exec(`UPDATE items SET storage=?, updated_at=? WHERE id=?`, storage, ts, id)
	if err != nil {
		return Item{}, err
	}
	label := "на баланс"
	if storage == StorageTemporary {
		label = "во временное хранение"
	}
	_ = s.addMovement(id, "storage", 0, old.Cell, old.Cell, label)
	return s.Get(id)
}
