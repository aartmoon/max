package gar

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"testing"
)

type captureStore struct {
	objects []AddressObject
	houses  []HouseRecord
	links   []HierarchyItem
	params  []Param
}

func (s *captureStore) UpsertAddressObjects(_ context.Context, items []AddressObject) error {
	s.objects = append(s.objects, items...)
	return nil
}

func (s *captureStore) UpsertHierarchy(_ context.Context, items []HierarchyItem) error {
	s.links = append(s.links, items...)
	return nil
}

func (s *captureStore) UpsertHouseTypes(_ context.Context, items []HouseType) error { return nil }

func (s *captureStore) UpsertParamTypes(_ context.Context, items []ParamType) error { return nil }

func (s *captureStore) UpsertParams(_ context.Context, items []Param) error {
	s.params = append(s.params, items...)
	return nil
}

func (s *captureStore) UpsertHouses(_ context.Context, items []HouseRecord) error {
	s.houses = append(s.houses, items...)
	return nil
}

func (s *captureStore) RefreshAddresses(context.Context) error { return nil }

func TestImportZipStreamsRequiredGARFiles(t *testing.T) {
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	addZipFile(t, zw, "AS_ADDR_OBJ_2_250_01_04_01_01.XML", `<ADDRESSOBJECTS>
<OBJECT ID="1" OBJECTID="10" OBJECTGUID="00000000-0000-0000-0000-000000000010" NAME="Татарстан" TYPENAME="Респ" LEVEL="1" ISACTUAL="1" ISACTIVE="1" UPDATEDATE="2026-09-01" STARTDATE="2020-01-01" ENDDATE="2079-06-06" CHANGEID="1" OPERTYPEID="10"/>
</ADDRESSOBJECTS>`)
	addZipFile(t, zw, "AS_ADM_HIERARCHY_2_250_04_04_01_01.XML", `<ITEMS>
<ITEM ID="1" OBJECTID="100" PARENTOBJID="10" CHANGEID="1" ISACTIVE="1" UPDATEDATE="2026-09-01" STARTDATE="2020-01-01" ENDDATE="2079-06-06"/>
</ITEMS>`)
	addZipFile(t, zw, "AS_HOUSES_2_250_08_04_01_01.XML", `<HOUSES>
<HOUSE ID="1" OBJECTID="100" OBJECTGUID="11111111-2222-3333-4444-555555555555" CHANGEID="1" HOUSENUM="20" HOUSETYPE="2" OPERTYPEID="10" UPDATEDATE="2026-09-01" STARTDATE="2020-01-01" ENDDATE="2079-06-06" ISACTUAL="1" ISACTIVE="1"/>
</HOUSES>`)
	addZipFile(t, zw, "AS_HOUSES_PARAMS_2_250_02_04_01_01.XML", `<PARAMS>
<PARAM ID="1" OBJECTID="100" CHANGEIDEND="0" TYPEID="5" VALUE="420000" UPDATEDATE="2026-09-01" STARTDATE="2020-01-01" ENDDATE="2079-06-06"/>
</PARAMS>`)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	store := &captureStore{}

	err := ImportZip(context.Background(), bytes.NewReader(b.Bytes()), int64(b.Len()), store, Options{BatchSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.objects) != 1 || store.objects[0].ObjectID != 10 {
		t.Fatalf("address objects were not imported: %+v", store.objects)
	}
	if len(store.links) != 1 || store.links[0].ParentObjectID == nil || *store.links[0].ParentObjectID != 10 {
		t.Fatalf("hierarchy was not imported: %+v", store.links)
	}
	if len(store.houses) != 1 || store.houses[0].ObjectGUID != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("houses were not imported: %+v", store.houses)
	}
	if len(store.params) != 1 || store.params[0].Value != "420000" {
		t.Fatalf("house params were not imported: %+v", store.params)
	}
}

func TestImportZipRejectsArchiveWithoutSupportedXMLFiles(t *testing.T) {
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	addZipFile(t, zw, "16_HOUSE.FI", "not xml gar")
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	err := ImportZip(context.Background(), bytes.NewReader(b.Bytes()), int64(b.Len()), &captureStore{}, Options{BatchSize: 2})
	if !errors.Is(err, ErrNoSupportedFiles) {
		t.Fatalf("expected ErrNoSupportedFiles, got %v", err)
	}
}

func addZipFile(t *testing.T, zw *zip.Writer, name, body string) {
	t.Helper()
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
}
