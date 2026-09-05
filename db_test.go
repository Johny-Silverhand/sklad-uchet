package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"  Винт  M6  ": "винт m6",
		"ВИНТ\tM6":    "винт m6",
		"A":           "a",
	}
	for in, want := range cases {
		if got := NormalizeName(in); got != want {
			t.Fatalf("NormalizeName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCRUDAndMerge(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("APPDATA", filepath.Join(tmp, "AppData"))
	_ = os.MkdirAll(filepath.Join(tmp, "AppData"), 0o755)

	store, err := OpenStore()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Fresh DB must be empty — no demo seed
	all, err := store.List(ListFilter{})
	if err != nil || len(all) != 0 {
		t.Fatalf("expected empty DB, got %d err=%v", len(all), err)
	}

	th, err := store.CreateTheme("Крепёж", 1)
	if err != nil {
		t.Fatal(err)
	}
	tid := th.ID

	a, dups, err := store.Create(UpsertInput{
		Name: "Винт M6", Kind: KindZapchast, Quantity: 10, MinQty: 2, Cell: "A-1", SKU: "V1",
		Storage: StorageBalance, ThemeID: &tid,
	})
	if err != nil || len(dups) != 0 {
		t.Fatalf("create a: %v dups=%v", err, dups)
	}
	if a.Storage != StorageBalance || a.ThemeID == nil || *a.ThemeID != tid {
		t.Fatalf("storage/theme: %+v", a)
	}
	if a.MinQty != 2 {
		t.Fatalf("min_qty=%d", a.MinQty)
	}
	_, dups, err = store.Create(UpsertInput{Name: "винт  m6", Kind: KindZapchast, Quantity: 5, Cell: "B-3", Notes: "коробка"})
	if err == nil || err.Error() != "duplicate" || len(dups) != 1 {
		t.Fatalf("expected duplicate warn, got err=%v dups=%v", err, dups)
	}
	b, _, err := store.Create(UpsertInput{
		Name: "винт  m6", Kind: KindZapchast, Quantity: 5, MinQty: 5, Cell: "B-3", Notes: "коробка",
		Storage: StorageTemporary, Force: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.Storage != StorageTemporary {
		t.Fatalf("temp storage=%s", b.Storage)
	}
	moved, err := store.SetStorage(b.ID, StorageBalance)
	if err != nil || moved.Storage != StorageBalance {
		t.Fatalf("set storage: %v %+v", err, moved)
	}
	merged, err := store.Merge(a.ID, []int64{b.ID})
	if err != nil {
		t.Fatal(err)
	}
	if merged.Quantity != 15 {
		t.Fatalf("qty=%d", merged.Quantity)
	}
	if merged.MinQty != 5 {
		t.Fatalf("min after merge=%d", merged.MinQty)
	}
	if merged.Cell != "A-1" {
		t.Fatalf("cell=%s", merged.Cell)
	}
	if merged.SKU != "V1" {
		t.Fatalf("sku=%s", merged.SKU)
	}
	if merged.Notes == "" || !strings.Contains(merged.Notes, "Также: B-3") {
		t.Fatalf("notes=%q", merged.Notes)
	}
	groups, err := store.DuplicateGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected no dups, got %d", len(groups))
	}

	adj, err := store.Adjust(merged.ID, -3)
	if err != nil || adj.Quantity != 12 {
		t.Fatalf("adjust: %v qty=%d", err, adj.Quantity)
	}
	cellMoved, err := store.MoveToCell(merged.ID, "C-9")
	if err != nil || cellMoved.Cell != "C-9" {
		t.Fatalf("move: %v cell=%s", err, cellMoved.Cell)
	}
	st, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if st.TotalItems != 1 || st.TotalQty != 12 {
		t.Fatalf("stats=%+v", st)
	}
	if st.ByStorage[StorageBalance] != 1 {
		t.Fatalf("by_storage=%v", st.ByStorage)
	}
	low, err := store.List(ListFilter{LowStock: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(low) != 0 {
		t.Fatalf("expected no low, got %d", len(low))
	}
	_, _ = store.Adjust(merged.ID, -10)
	low, err = store.List(ListFilter{LowStock: true})
	if err != nil || len(low) != 1 {
		t.Fatalf("low stock expected 1, got %d err=%v", len(low), err)
	}

	bal, err := store.List(ListFilter{Storage: StorageBalance, ThemeOnly: true, ThemeID: &tid})
	if err != nil || len(bal) != 1 {
		t.Fatalf("theme filter: %d err=%v", len(bal), err)
	}

	if err := store.DeleteTheme(tid); err != nil {
		t.Fatal(err)
	}
	it, err := store.Get(merged.ID)
	if err != nil || it.ThemeID != nil {
		t.Fatalf("theme should be cleared: %+v err=%v", it, err)
	}
}

func TestCSVRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("APPDATA", filepath.Join(tmp, "AppData"))
	store, err := OpenStore()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, _, err = store.Create(UpsertInput{Name: "Гайка", Kind: KindKomplektuyushchee, Quantity: 3, MinQty: 1, Cell: "D-1", Storage: StorageBalance, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := store.ExportCSV(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Гайка") || !strings.Contains(buf.String(), "storage") {
		t.Fatalf("csv missing: %s", buf.String())
	}
	res, err := store.ImportCSV(strings.NewReader("name;kind;quantity;min_qty;cell;sku;notes;storage;theme\nБолт;zapchast;7;2;E-1;B1;;temporary;Метизы\n"), true)
	if err != nil || res.Created != 1 {
		t.Fatalf("import: %+v err=%v", res, err)
	}
	items, _ := store.List(ListFilter{Storage: StorageTemporary})
	if len(items) != 1 || items[0].ThemeName != "Метизы" {
		t.Fatalf("imported temp/theme: %+v", items)
	}
}

func TestMigrateOldItemsWithoutStorage(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "old.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
CREATE TABLE schema_version (version INTEGER NOT NULL);
INSERT INTO schema_version(version) VALUES (1);
CREATE TABLE items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  name_norm TEXT NOT NULL,
  kind TEXT NOT NULL,
  quantity INTEGER NOT NULL DEFAULT 0,
  cell TEXT NOT NULL DEFAULT '',
  sku TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
INSERT INTO items(name,name_norm,kind,quantity,cell,sku,notes,created_at,updated_at)
VALUES ('Болт','болт','zapchast',2,'A-1','','','2020-01-01T00:00:00Z','2020-01-01T00:00:00Z');
`)
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	s := &Store{}
	db2, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	s.db = db2
	s.dbPath = dbPath
	if err := s.migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	var storage string
	if err := db2.QueryRow(`SELECT storage FROM items WHERE name='Болт'`).Scan(&storage); err != nil {
		t.Fatalf("select storage: %v", err)
	}
	if storage != StorageBalance {
		t.Fatalf("storage=%q", storage)
	}
	_ = db2.Close()
}
