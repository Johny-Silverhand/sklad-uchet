package main

import (
	"fmt"
	"strings"
)

func (s *Store) ListThemes() ([]Theme, error) {
	rows, err := s.db.Query(`
SELECT t.id, t.name, t.sort_order, t.created_at,
       (SELECT COUNT(*) FROM items i WHERE i.theme_id = t.id) AS item_count
FROM themes t
ORDER BY t.sort_order ASC, t.name ASC, t.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Theme
	for rows.Next() {
		var th Theme
		if err := rows.Scan(&th.ID, &th.Name, &th.SortOrder, &th.CreatedAt, &th.ItemCount); err != nil {
			return nil, err
		}
		out = append(out, th)
	}
	if out == nil {
		out = []Theme{}
	}
	return out, rows.Err()
}

func (s *Store) CreateTheme(name string, sortOrder int) (Theme, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Theme{}, fmt.Errorf("укажите название темы")
	}
	if strings.EqualFold(name, "Без темы") {
		return Theme{}, fmt.Errorf("это служебное название")
	}
	ts := nowISO()
	res, err := s.db.Exec(`INSERT INTO themes (name, sort_order, created_at) VALUES (?, ?, ?)`, name, sortOrder, ts)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Theme{}, fmt.Errorf("тема с таким названием уже есть")
		}
		return Theme{}, err
	}
	id, _ := res.LastInsertId()
	return s.GetTheme(id)
}

func (s *Store) GetTheme(id int64) (Theme, error) {
	var th Theme
	err := s.db.QueryRow(`
SELECT t.id, t.name, t.sort_order, t.created_at,
       (SELECT COUNT(*) FROM items i WHERE i.theme_id = t.id)
FROM themes t WHERE t.id = ?`, id).Scan(&th.ID, &th.Name, &th.SortOrder, &th.CreatedAt, &th.ItemCount)
	if err != nil {
		return Theme{}, fmt.Errorf("тема не найдена")
	}
	return th, nil
}

func (s *Store) UpdateTheme(id int64, name string, sortOrder int) (Theme, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Theme{}, fmt.Errorf("укажите название темы")
	}
	if strings.EqualFold(name, "Без темы") {
		return Theme{}, fmt.Errorf("это служебное название")
	}
	res, err := s.db.Exec(`UPDATE themes SET name=?, sort_order=? WHERE id=?`, name, sortOrder, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Theme{}, fmt.Errorf("тема с таким названием уже есть")
		}
		return Theme{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Theme{}, fmt.Errorf("тема не найдена")
	}
	return s.GetTheme(id)
}

func (s *Store) DeleteTheme(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`UPDATE items SET theme_id = NULL WHERE theme_id = ?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM themes WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("тема не найдена")
	}
	return tx.Commit()
}
