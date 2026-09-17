package repository

import (
	"reflect"
	"testing"
)

func TestSearchTokensNormalizePunctuationCaseAndWhitespace(t *testing.T) {
	got := SearchTokens("  Казань,   Чистопольская д. 20 ")
	want := []string{"казань", "чистопольская", "д", "20"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens mismatch\nwant: %#v\n got: %#v", want, got)
	}
}
