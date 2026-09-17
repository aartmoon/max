package repository

import (
	"reflect"
	"strings"
	"testing"

	"tvoydom/domain"
)

func TestSearchTokensNormalizePunctuationCaseAndWhitespace(t *testing.T) {
	got := SearchTokens("  Казань,   Чистопольская д. 20 ")
	want := []string{"казань", "чистопольская", "д", "20"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestBuildAddressSearchQueryAppliesKindParentTokensAndLimit(t *testing.T) {
	parent := int64(100)
	sql, args := buildAddressSearchQuery(domain.AddressSearch{Query: "  Тверская,  12 ", Kind: "house", ParentObjectID: &parent, Limit: 7})
	for _, fragment := range []string{"object_kind=$1", "parent_object_id=$2", "search_text LIKE $3", "search_text LIKE $4", "LIMIT $6"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("query missing %q:\n%s", fragment, sql)
		}
	}
	want := []any{"house", int64(100), "%тверская%", "%12%", "тверская 12", 7}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args\nwant: %#v\n got: %#v", want, args)
	}
}

func TestBuildAddressSearchQueryAllowsEmptyQueryForChildren(t *testing.T) {
	parent := int64(200)
	_, args := buildAddressSearchQuery(domain.AddressSearch{Kind: "apartment", ParentObjectID: &parent, Limit: 10})
	want := []any{"apartment", int64(200), "", 10}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args\nwant: %#v\n got: %#v", want, args)
	}
}
