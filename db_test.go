package main

import (
	"os"
	"path/filepath"
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

	a, dups, err := store.Create(UpsertInput{Name: "Винт M6", Kind: KindZapchast, Quantity: 10, Cell: "A-1", SKU: "V1"})
	if err != nil || len(dups) != 0 {
		t.Fatalf("create a: %v dups=%v", err, dups)
	}
	_, dups, err = store.Create(UpsertInput{Name: "винт  m6", Kind: KindZapchast, Quantity: 5, Cell: "B-3", Notes: "коробка"})
	if err == nil || err.Error() != "duplicate" || len(dups) != 1 {
		t.Fatalf("expected duplicate warn, got err=%v dups=%v", err, dups)
	}
	b, _, err := store.Create(UpsertInput{Name: "винт  m6", Kind: KindZapchast, Quantity: 5, Cell: "B-3", Notes: "коробка", Force: true})
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
	if merged.Cell != "A-1" {
		t.Fatalf("cell=%s", merged.Cell)
	}
	if merged.SKU != "V1" {
		t.Fatalf("sku=%s", merged.SKU)
	}
	if merged.Notes == "" || !contains(merged.Notes, "Также: B-3") {
		t.Fatalf("notes=%q", merged.Notes)
	}
	groups, err := store.DuplicateGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected no dups, got %d", len(groups))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
