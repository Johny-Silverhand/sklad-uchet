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
	KindZapchast          = "zapchast"
	KindUstroystvo        = "ustroystvo"
	KindKomplektuyushchee = "komplektuyushchee"

	StorageBalance   = "balance"
	StorageTemporary = "temporary"

	schemaVersion = 3
)

type Theme struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	ItemCount int    `json:"item_count"`
}

type Item struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Quantity  int    `json:"quantity"`
	MinQty    int    `json:"min_qty"`
	Cell      string `json:"cell"`
	SKU       string `json:"sku"`
	Notes     string `json:"notes"`
	Storage   string `json:"storage"`
	ThemeID   *int64 `json:"theme_id"`
	ThemeName string `json:"theme_name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	LowStock  bool   `json:"low_stock"`
}

type DuplicateGroup struct {
	NormalizedName string `json:"normalized_name"`
	Items          []Item `json:"items"`
	TotalQty       int    `json:"total_qty"`
}

type Movement struct {
	ID        int64  `json:"id"`
	ItemID    int64  `json:"item_id"`
	Kind      string `json:"kind"`
	Delta     int    `json:"delta"`
	FromCell  string `json:"from_cell"`
	ToCell    string `json:"to_cell"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
}

type CellStat struct {
	Cell  string `json:"cell"`
	Count int    `json:"count"`
	Qty   int    `json:"qty"`
}

type Stats struct {
	TotalItems   int            `json:"total_items"`
	TotalQty     int            `json:"total_qty"`
	LowStock     int            `json:"low_stock"`
	ByKind       map[string]int `json:"by_kind"`
	QtyByKind    map[string]int `json:"qty_by_kind"`
	ByStorage    map[string]int `json:"by_storage"`
	QtyByStorage map[string]int `json:"qty_by_storage"`
	TopCells     []CellStat     `json:"top_cells"`
}

type Store struct {
	db     *sql.DB
	dbPath string
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
	s := &Store{db: db, dbPath: dbPath}
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
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS themes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  name_norm TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('zapchast','ustroystvo','komplektuyushchee')),
  quantity INTEGER NOT NULL DEFAULT 0 CHECK(quantity >= 0),
  min_qty INTEGER NOT NULL DEFAULT 0 CHECK(min_qty >= 0),
  cell TEXT NOT NULL DEFAULT '',
  sku TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  storage TEXT NOT NULL DEFAULT 'balance' CHECK(storage IN ('balance','temporary')),
  theme_id INTEGER REFERENCES themes(id) ON DELETE SET NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_items_name_norm ON items(name_norm);
CREATE INDEX IF NOT EXISTS idx_items_cell ON items(cell);
CREATE INDEX IF NOT EXISTS idx_items_kind ON items(kind);
CREATE INDEX IF NOT EXISTS idx_items_storage ON items(storage);
CREATE INDEX IF NOT EXISTS idx_items_theme ON items(theme_id);
CREATE TABLE IF NOT EXISTS movements (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  item_id INTEGER NOT NULL,
  kind TEXT NOT NULL,
  delta INTEGER NOT NULL DEFAULT 0,
  from_cell TEXT NOT NULL DEFAULT '',
  to_cell TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_movements_item ON movements(item_id);
CREATE INDEX IF NOT EXISTS idx_movements_created ON movements(created_at);
`); err != nil {
		return err
	}

	var ver int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&ver)
	if err != nil {
		return err
	}

	if ver < 2 {
		var hasMin int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('items') WHERE name='min_qty'`).Scan(&hasMin)
		if hasMin == 0 {
			if _, err := s.db.Exec(`ALTER TABLE items ADD COLUMN min_qty INTEGER NOT NULL DEFAULT 0`); err != nil {
				return err
			}
		}
	}

	if ver < 3 {
		var hasStorage int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('items') WHERE name='storage'`).Scan(&hasStorage)
		if hasStorage == 0 {
			if _, err := s.db.Exec(`ALTER TABLE items ADD COLUMN storage TEXT NOT NULL DEFAULT 'balance'`); err != nil {
				return err
			}
		}
		var hasTheme int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('items') WHERE name='theme_id'`).Scan(&hasTheme)
		if hasTheme == 0 {
			if _, err := s.db.Exec(`ALTER TABLE items ADD COLUMN theme_id INTEGER REFERENCES themes(id) ON DELETE SET NULL`); err != nil {
				return err
			}
		}
		if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_items_storage ON items(storage)`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_items_theme ON items(theme_id)`); err != nil {
			return err
		}
		// Never seed demo items. Existing DBs keep user data; fresh DBs stay empty.
	}

	if ver < schemaVersion {
		if _, err := s.db.Exec(`DELETE FROM schema_version`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version(version) VALUES (?)`, schemaVersion); err != nil {
			return err
		}
	}
	return nil
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

func validStorage(st string) bool {
	switch st {
	case StorageBalance, StorageTemporary:
		return true
	}
	return false
}

func scanItem(row interface{ Scan(dest ...any) error }) (Item, error) {
	var it Item
	var themeID sql.NullInt64
	var themeName sql.NullString
	err := row.Scan(
		&it.ID, &it.Name, &it.Kind, &it.Quantity, &it.MinQty, &it.Cell, &it.SKU, &it.Notes,
		&it.Storage, &themeID, &themeName, &it.CreatedAt, &it.UpdatedAt,
	)
	if err != nil {
		return it, err
	}
	if it.Storage == "" {
		it.Storage = StorageBalance
	}
	if themeID.Valid {
		id := themeID.Int64
		it.ThemeID = &id
		it.ThemeName = themeName.String
	} else {
		it.ThemeName = "Без темы"
	}
	it.LowStock = it.MinQty > 0 && it.Quantity <= it.MinQty
	return it, nil
}

const itemCols = `i.id, i.name, i.kind, i.quantity, i.min_qty, i.cell, i.sku, i.notes, i.storage, i.theme_id, COALESCE(t.name, ''), i.created_at, i.updated_at`
const itemFrom = `items i LEFT JOIN themes t ON t.id = i.theme_id`
