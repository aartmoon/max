package gishousing

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tvoydom/domain"
)

func TestClientFetchHouse(t *testing.T) {
	const fias = "11111111-2222-3333-4444-555555555555"
	const gis = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "searchByFiasHouseCodeList") {
			if !strings.Contains(r.URL.Path, fias) || r.URL.Query().Get("useReadOnlyDataSource") != "true" {
				t.Fatalf("unexpected lookup request: %s", r.URL.String())
			}
			_, _ = w.Write([]byte(`{"houseList":[{"guid":"` + gis + `","houseHMGuid":"` + gis + `","houseType":{"code":"1"},"house":{"code":"` + fias + `"},"entranceCount":4,"residentialPremiseCount":120,"managementOrganization":{"shortName":"ООО УК Дом","phone":"+7 495 000-00-00"}}]}`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/1/"+gis) {
			t.Fatalf("unexpected detail request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"guid":"` + gis + `","houseType":{"code":"1"},"address":{"house":{"houseGuid":"` + fias + `"}},"cadastreNumber":"77:01:0000000:1","totalSquare":"12345.6","residentialSquare":"9876.5","floorCountMax":16,"entranceCount":4,"residentialPremiseCount":120,"buildingYear":"1987","managementOrganization":{"shortName":"ООО УК Дом","phone":"+7 495 000-00-00"}}`))
	}))
	defer server.Close()

	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	client := NewClient(server.URL, time.Second, server.Client())
	client.Now = func() time.Time { return now }
	profile, err := client.FetchHouse(context.Background(), fias)
	if err != nil {
		t.Fatal(err)
	}
	if profile.GISHouseGUID != gis || profile.GISHouseType != "1" || profile.CadastralNumber == nil || *profile.CadastralNumber != "77:01:0000000:1" {
		t.Fatalf("unexpected identity: %+v", profile)
	}
	if profile.TotalArea == nil || *profile.TotalArea != 12345.6 || profile.LivingArea == nil || *profile.LivingArea != 9876.5 || profile.Floors == nil || *profile.Floors != 16 || profile.Apartments == nil || *profile.Apartments != 120 {
		t.Fatalf("unexpected characteristics: %+v", profile)
	}
	if profile.Organization == nil || *profile.Organization != "ООО УК Дом" || profile.Contact == nil || *profile.Contact != "+7 495 000-00-00" || profile.FetchedAt != now || len(profile.RawPayload) == 0 {
		t.Fatalf("unexpected metadata: %+v", profile)
	}
}

func TestClientErrors(t *testing.T) {
	cases := map[string]struct {
		handler http.HandlerFunc
		want    error
	}{
		"empty":     {func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"houseList":[]}`)) }, domain.ErrHouseProfileNotFound},
		"malformed": {func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{`)) }, domain.ErrHouseProfileUnavailable},
		"status":    {func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "bad", http.StatusBadGateway) }, domain.ErrHouseProfileUnavailable},
		"oversized": {func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", maxResponseBytes+1)))
		}, domain.ErrHouseProfileUnavailable},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()
			_, err := NewClient(server.URL, time.Second, server.Client()).FetchHouse(context.Background(), "11111111-2222-3333-4444-555555555555")
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}
