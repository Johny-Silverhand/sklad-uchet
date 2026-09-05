package main

import (
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

	a, dups, err := store.Create(UpsertInput{Name: "Винт M6", Kind: KindZapchast, Quantity: 10, MinQty: 2, Cell: "A-1", SKU: "V1"})
	if err != nil || len(dups) != 0 {
		t.Fatalf("create a: %v dups=%v", err, dups)
	}
	if a.MinQty != 2 {
		t.Fatalf("min_qty=%d", a.MinQty)
	}
	_, dups, err = store.Create(UpsertInput{Name: "винт  m6", Kind: KindZapchast, Quantity: 5, Cell: "B-3", Notes: "коробка"})
	if err == nil || err.Error() != "duplicate" || len(dups) != 1 {
		t.Fatalf("expected duplicate warn, got err=%v dups=%v", err, dups)
	}
	b, _, err := store.Create(UpsertInput{Name: "винт  m6", Kind: KindZapchast, Quantity: 5, MinQty: 5, Cell: "B-3", Notes: "коробка", Force: true})
	if err != nil {
		t.Fatal(err)
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
	moved, err := store.MoveToCell(merged.ID, "C-9")
	if err != nil || moved.Cell != "C-9" {
		t.Fatalf("move: %v cell=%s", err, moved.Cell)
	}
	st, err := store.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if st.TotalItems != 1 || st.TotalQty != 12 {
		t.Fatalf("stats=%+v", st)
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
	_, _, err = store.Create(UpsertInput{Name: "Гайка", Kind: KindKomplektuyushchee, Quantity: 3, MinQty: 1, Cell: "D-1", Force: true})
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := store.ExportCSV(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Гайка") {
		t.Fatalf("csv missing name: %s", buf.String())
	}
	res, err := store.ImportCSV(strings.NewReader("name;kind;quantity;min_qty;cell;sku;notes\nБолт;zapchast;7;2;E-1;B1;\n"), true)
	if err != nil || res.Created != 1 {
		t.Fatalf("import: %+v err=%v", res, err)
	}
}
