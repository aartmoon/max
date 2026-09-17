package importer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSortsFilesByManifestOrder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range allFixtureNames() {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("<ROOT/>"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	files, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if files[0].Family.Key != "address_objects" || files[len(files)-1].Family.Key != "reestr_objects" {
		t.Fatalf("unexpected order: first=%s last=%s", files[0].Family.Key, files[len(files)-1].Family.Key)
	}
}

func TestDiscoverDistinguishesEmptyAndIncompleteInput(t *testing.T) {
	dir := t.TempDir()
	_, err := Discover(dir)
	if !errors.Is(err, ErrNoSupportedFiles) {
		t.Fatalf("expected ErrNoSupportedFiles, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AS_HOUSES_test.XML"), []byte("<HOUSES/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Discover(dir)
	if !errors.Is(err, ErrIncompleteSnapshot) {
		t.Fatalf("expected ErrIncompleteSnapshot, got %v", err)
	}
}

func TestDiscoverRejectsDuplicateFamily(t *testing.T) {
	dir := t.TempDir()
	for _, name := range allFixtureNames() {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "AS_HOUSES_duplicate.XML"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Discover(dir)
	if !errors.Is(err, ErrDuplicateFamily) {
		t.Fatalf("expected ErrDuplicateFamily, got %v", err)
	}
}

func allFixtureNames() []string {
	return []string{
		"AS_REESTR_OBJECTS_test.XML", "AS_NORMATIVE_DOCS_test.XML", "AS_CHANGE_HISTORY_test.XML",
		"AS_STEADS_PARAMS_test.XML", "AS_STEADS_test.XML", "AS_ROOMS_PARAMS_test.XML", "AS_ROOMS_test.XML",
		"AS_CARPLACES_PARAMS_test.XML", "AS_CARPLACES_test.XML", "AS_APARTMENTS_PARAMS_test.XML",
		"AS_APARTMENTS_test.XML", "AS_HOUSES_PARAMS_test.XML", "AS_HOUSES_test.XML",
		"AS_MUN_HIERARCHY_test.XML", "AS_ADM_HIERARCHY_test.XML", "AS_ADDR_OBJ_DIVISION_test.XML",
		"AS_ADDR_OBJ_PARAMS_test.XML", "AS_ADDR_OBJ_test.XML",
	}
}
