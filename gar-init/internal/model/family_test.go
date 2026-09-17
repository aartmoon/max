package model

import "testing"

func TestFamiliesContainEverySupportedXMLType(t *testing.T) {
	want := []string{
		"address_objects", "addr_object_params", "addr_object_divisions", "adm_hierarchy",
		"mun_hierarchy", "houses", "house_params", "apartments", "apartment_params",
		"carplaces", "carplace_params", "rooms", "room_params", "steads", "stead_params",
		"change_history", "normative_docs", "reestr_objects",
	}
	got := Families()
	if len(got) != len(want) {
		t.Fatalf("want %d families, got %d", len(want), len(got))
	}
	for index, key := range want {
		if got[index].Key != key {
			t.Fatalf("family %d: want %q, got %q", index, key, got[index].Key)
		}
	}
}

func TestMatchFileUsesMostSpecificPrefix(t *testing.T) {
	tests := map[string]string{
		"AS_ADDR_OBJ_PARAMS_20260910_id.XML":   "addr_object_params",
		"AS_ADDR_OBJ_DIVISION_20260910_id.XML": "addr_object_divisions",
		"AS_ADDR_OBJ_20260910_id.XML":          "address_objects",
		"AS_HOUSES_PARAMS_20260910_id.XML":     "house_params",
		"AS_HOUSES_20260910_id.XML":            "houses",
		"as_apartments_params_20260910_id.xml": "apartment_params",
	}
	for name, want := range tests {
		family, ok := MatchFile(name)
		if !ok || family.Key != want {
			t.Errorf("%s: want %s, got %+v, %v", name, want, family, ok)
		}
	}
}

func TestMatchFileRejectsNonXMLAndUnknownFiles(t *testing.T) {
	for _, name := range []string{"AS_HOUSES_test.zip", "README.XML", "AS_HOUSES.XML.bak"} {
		if family, ok := MatchFile(name); ok {
			t.Errorf("unexpected match for %s: %+v", name, family)
		}
	}
}
