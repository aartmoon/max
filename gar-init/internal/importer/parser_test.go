package importer

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"tvoydom/gar-init/internal/model"
)

func TestParseStreamsHouseAttributesInBoundedBatches(t *testing.T) {
	xmlData := `<HOUSES><HOUSE OBJECTID="10" OBJECTGUID="11111111-2222-3333-4444-555555555555" HOUSENUM="7" BUILDNUM="2" ISACTUAL="1" ISACTIVE="true" UPDATEDATE="2026-09-10"/><HOUSE OBJECTID="11" HOUSENUM="8" ISACTUAL="0" ISACTIVE="1"/></HOUSES>`
	var sizes []int
	var first []any
	count, err := Parse(context.Background(), strings.NewReader(xmlData), mustFamily(t, "houses"), 1, 100000,
		func(_ context.Context, _ model.Family, rows [][]any) error {
			sizes = append(sizes, len(rows))
			if first == nil {
				first = append([]any(nil), rows[0]...)
			}
			return nil
		}, func(int64) {})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || !reflect.DeepEqual(sizes, []int{1, 1}) {
		t.Fatalf("count=%d sizes=%v", count, sizes)
	}
	family := mustFamily(t, "houses")
	values := rowByColumn(family, first)
	if values["object_id"] != int64(10) || values["object_guid"] != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("unexpected identifiers: %#v", values)
	}
	if values["is_actual"] != true || values["is_active"] != true {
		t.Fatalf("unexpected booleans: %#v", values)
	}
	if got, ok := values["update_date"].(time.Time); !ok || got.Format(time.DateOnly) != "2026-09-10" {
		t.Fatalf("unexpected date: %#v", values["update_date"])
	}
	raw, ok := values["raw_attributes"].([]byte)
	if !ok || !strings.Contains(string(raw), `"BUILDNUM":"2"`) {
		t.Fatalf("raw attributes missing BUILDNUM: %T %s", values["raw_attributes"], raw)
	}
}

func TestParseWritesFinalPartialBatchAndReportsProgress(t *testing.T) {
	input := `<PARAMS><PARAM OBJECTID="1" TYPEID="2" VALUE="a"/><PARAM OBJECTID="2" TYPEID="2" VALUE="b"/><PARAM OBJECTID="3" TYPEID="2" VALUE="c"/></PARAMS>`
	var sizes []int
	var progress []int64
	count, err := Parse(context.Background(), strings.NewReader(input), mustFamily(t, "house_params"), 2, 2,
		func(_ context.Context, _ model.Family, rows [][]any) error {
			sizes = append(sizes, len(rows))
			return nil
		}, func(n int64) { progress = append(progress, n) })
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 || !reflect.DeepEqual(sizes, []int{2, 1}) || !reflect.DeepEqual(progress, []int64{2}) {
		t.Fatalf("count=%d sizes=%v progress=%v", count, sizes, progress)
	}
}

func TestParseAcceptsFamilySpecificChildNames(t *testing.T) {
	tests := []struct {
		family string
		xml    string
	}{
		{"address_objects", `<ROOT><OBJECT OBJECTID="1"/></ROOT>`},
		{"adm_hierarchy", `<ROOT><ITEM OBJECTID="1" PARENTOBJID="2"/></ROOT>`},
		{"apartments", `<ROOT><APARTMENT OBJECTID="1" NUMBER="5"/></ROOT>`},
		{"rooms", `<ROOT><ROOM OBJECTID="1" NUMBER="2"/></ROOT>`},
		{"normative_docs", `<ROOT><DOCUMENT ID="1" NAME="doc"/></ROOT>`},
	}
	for _, tt := range tests {
		t.Run(tt.family, func(t *testing.T) {
			count, err := Parse(context.Background(), strings.NewReader(tt.xml), mustFamily(t, tt.family), 10, 10, noOpWriter, func(int64) {})
			if err != nil || count != 1 {
				t.Fatalf("count=%d err=%v", count, err)
			}
		})
	}
}

<<<<<<< HEAD
func TestParseChangeHistoryAcceptsAddressObjectUUID(t *testing.T) {
	input := `<ITEM CHANGEID="10" OBJECTID="20" ADROBJECTID="7f89b3a2-70af-4957-9395-c72f882e56f5" OPERTYPEID="30"/>`
	var row []any
	count, err := Parse(context.Background(), strings.NewReader(input), mustFamily(t, "change_history"), 10, 100,
		func(_ context.Context, _ model.Family, rows [][]any) error {
			row = append([]any(nil), rows[0]...)
			return nil
		}, func(int64) {})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("want 1 row, got %d", count)
	}
	values := rowByColumn(mustFamily(t, "change_history"), row)
	if got := values["address_object_id"]; got != "7f89b3a2-70af-4957-9395-c72f882e56f5" {
		t.Fatalf("unexpected ADROBJECTID: %#v", got)
	}
}

=======
>>>>>>> codex/gar-init
func TestParseRejectsMalformedTypedAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attr  string
		value string
	}{
		{"integer", "OBJECTID", "not-a-number"},
		{"uuid", "OBJECTGUID", "not-a-uuid"},
		{"date", "UPDATEDATE", "10/09/2026"},
		{"boolean", "ISACTIVE", "yes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := `<HOUSES><HOUSE ` + tt.attr + `="` + tt.value + `"/></HOUSES>`
			_, err := Parse(context.Background(), strings.NewReader(input), mustFamily(t, "houses"), 10, 100, noOpWriter, func(int64) {})
			if err == nil || !strings.Contains(err.Error(), tt.attr) || !strings.Contains(err.Error(), "offset") {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestParseReturnsXMLAndWriterErrors(t *testing.T) {
	_, err := Parse(context.Background(), strings.NewReader(`<HOUSES><HOUSE OBJECTID="1"></HOUSES>`), mustFamily(t, "houses"), 10, 100, noOpWriter, func(int64) {})
	if err == nil || !strings.Contains(err.Error(), "offset") {
		t.Fatalf("expected XML offset error, got %v", err)
	}
	errBoom := errors.New("copy failed")
	_, err = Parse(context.Background(), strings.NewReader(`<HOUSES><HOUSE OBJECTID="1"/></HOUSES>`), mustFamily(t, "houses"), 1, 100,
		func(context.Context, model.Family, [][]any) error { return errBoom }, func(int64) {})
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected writer error, got %v", err)
	}
}

func TestParseHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Parse(ctx, strings.NewReader(`<HOUSES><HOUSE OBJECTID="1"/></HOUSES>`), mustFamily(t, "houses"), 10, 100, noOpWriter, func(int64) {})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func mustFamily(t *testing.T, key string) model.Family {
	t.Helper()
	for _, family := range model.Families() {
		if family.Key == key {
			return family
		}
	}
	t.Fatalf("unknown family %s", key)
	return model.Family{}
}

func noOpWriter(context.Context, model.Family, [][]any) error { return nil }

func rowByColumn(family model.Family, row []any) map[string]any {
	values := make(map[string]any, len(row))
	for index, column := range family.Columns() {
		values[column] = row[index]
	}
	return values
}
