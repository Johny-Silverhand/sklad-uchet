package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	_ "modernc.org/sqlite"
)

const (
	KindZapchast         = "zapchast"
	KindUstroystvo       = "ustroystvo"
	KindKomplektuyushchee = "komplektuyushchee"
)


type Item struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Quantity  int    `json:"quantity"`
	Cell      string `json:"cell"`
	SKU       string `json:"sku"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type DuplicateGroup struct {
	NormalizedName string `json:"normalized_name"`
	Items          []Item `json:"items"`
	TotalQty       int    `json:"total_qty"`
}

type Store struct {
	db *sql.DB
}

func dataDir() (string, error) {
	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Roaming")
		}
	} else {
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".local", "share")
		}
	}
	dir := filepath.Join(base, "VictimokLabs", "SkladUchet")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func OpenStore() (*Store, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dir, "sklad.db")
	db, err := sql.Open("sqlite", dbPath)
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
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  name_norm TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('zapchast','ustroystvo','komplektuyushchee')),
  quantity INTEGER NOT NULL DEFAULT 0 CHECK(quantity >= 0),
  cell TEXT NOT NULL DEFAULT '',
  sku TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_items_name_norm ON items(name_norm);
CREATE INDEX IF NOT EXISTS idx_items_cell ON items(cell);
CREATE INDEX IF NOT EXISTS idx_items_kind ON items(kind);
`)
	return err
}

func NormalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	var b strings.Builder
	prevSpace := false
	for _, r := range name {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func validKind(k string) bool {
	switch k {
	case KindZapchast, KindUstroystvo, KindKomplektuyushchee:
		return true
	}
	return false
}

func scanItem(row interface{ Scan(dest ...any) error }) (Item, error) {
	var it Item
	err := row.Scan(&it.ID, &it.Name, &it.Kind, &it.Quantity, &it.Cell, &it.SKU, &it.Notes, &it.CreatedAt, &it.UpdatedAt)
	return it, err
}

const itemCols = `id, name, kind, quantity, cell, sku, notes, created_at, updated_at`

