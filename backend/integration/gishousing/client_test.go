package gishousing

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tvoydom/domain"
)

//go:embed testdata/arbat_lookup.json
var arbatLookup []byte

//go:embed testdata/arbat_detail.json
var arbatDetail []byte

//go:embed testdata/arbat_square.json
var arbatSquare []byte

func TestClientFetchHouse(t *testing.T) {
	const fias = "151b9095-9f1d-4ab5-afc6-db971eed9d49"
	const gis = "105a84e6-01b5-4bbb-a26d-e5ec59bee321"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "searchByFiasHouseCodeList") {
			if !strings.Contains(r.URL.Path, fias) || r.URL.Query().Get("useReadOnlyDataSource") != "true" {
				t.Fatalf("unexpected lookup request: %s", r.URL.String())
			}
			_, _ = w.Write(arbatLookup)
			return
		}
		if strings.Contains(r.URL.Path, "get-house-square-data") {
			if !strings.HasSuffix(r.URL.Path, "/"+gis) {
				t.Fatalf("unexpected square request: %s", r.URL.String())
			}
			_, _ = w.Write(arbatSquare)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/1/"+gis) {
			t.Fatalf("unexpected detail request: %s", r.URL.String())
		}
		_, _ = w.Write(arbatDetail)
	}))
	defer server.Close()

	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	client := NewClient(server.URL, time.Second, server.Client())
	client.Now = func() time.Time { return now }
	profile, err := client.FetchHouse(context.Background(), fias)
	if err != nil {
		t.Fatal(err)
	}
	if profile.GISHouseGUID != gis || profile.GISHouseType != "1" || profile.CadastralNumber == nil || *profile.CadastralNumber != "77:01:0001046:1013" {
		t.Fatalf("unexpected identity: %+v", profile)
	}
	if profile.Characteristics.HouseType == nil || *profile.Characteristics.HouseType != "Многоквартирный" || profile.Characteristics.ProjectSeries == nil || *profile.Characteristics.ProjectSeries != "Индивидуальный проект" {
		t.Fatalf("unexpected characteristics: %+v", profile)
	}
	if profile.Characteristics.ResidentialPremises == nil || *profile.Characteristics.ResidentialPremises != 15 || profile.Characteristics.NonResidentialPremises == nil || *profile.Characteristics.NonResidentialPremises != 31 || profile.Characteristics.OwnersOrShares == nil || *profile.Characteristics.OwnersOrShares != 61 {
		t.Fatalf("unexpected square summary: %+v", profile.Characteristics)
	}
	if profile.Characteristics.Floors != nil || profile.Characteristics.Entrances != nil {
		t.Fatalf("unpublished values must stay nil: %+v", profile.Characteristics)
	}
	if profile.Management.OGRN == nil || *profile.Management.OGRN != "5147746267906" || profile.Management.ContractStart == nil || profile.Management.ContractStart.Format("2006-01-02") != "2015-04-22" {
		t.Fatalf("unexpected management: %+v", profile.Management)
	}
	if profile.Apartments == nil || *profile.Apartments != 15 || profile.YearBuilt == nil || *profile.YearBuilt != 1870 || profile.Organization == nil || *profile.Organization != `ГБУ "Жилищник района Арбат"` {
		t.Fatalf("legacy compatibility missing: %+v", profile)
	}
	if !profile.SquareSummaryAvailable || len(profile.SquarePayload) == 0 || profile.FetchedAt != now || len(profile.RawPayload) == 0 {
		t.Fatalf("unexpected metadata: %+v", profile)
	}
	if len(profile.DataSources) != 2 || !profile.DataSources[0].Available || !profile.DataSources[1].Available {
		t.Fatalf("unexpected sources: %+v", profile.DataSources)
	}
}

func TestFloorAndPremiseFallbacks(t *testing.T) {
	const fias = "151b9095-9f1d-4ab5-afc6-db971eed9d49"
	details := []struct {
		name  string
		json  string
		floor int
		count int
	}{
		{"current", `{"guid":"gis","houseType":{"code":"1"},"floorCount":7,"floorCountMax":9,"residentialPremiseActualCount":12,"residentialPremiseConfirmedCount":11,"residentialPremiseCount":10}`, 7, 12},
		{"legacy", `{"guid":"gis","houseType":{"code":"1"},"floorCountMax":9,"residentialPremiseCount":10}`, 9, 10},
	}
	for _, tc := range details {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.Contains(r.URL.Path, "searchByFiasHouseCodeList"):
					_, _ = w.Write([]byte(`{"houseList":[{"guid":"gis","houseType":{"code":"1"},"house":{"code":"` + fias + `"}}]}`))
				case strings.Contains(r.URL.Path, "get-house-square-data"):
					http.Error(w, "unavailable", http.StatusBadGateway)
				default:
					_, _ = w.Write([]byte(tc.json))
				}
			}))
			defer server.Close()
			profile, err := NewClient(server.URL, time.Second, server.Client()).FetchHouse(context.Background(), fias)
			if err != nil {
				t.Fatal(err)
			}
			if profile.Floors == nil || *profile.Floors != tc.floor || profile.Apartments == nil || *profile.Apartments != tc.count {
				t.Fatalf("unexpected fallbacks: %+v", profile)
			}
		})
	}
}

func TestOperationYearDoesNotBecomeConstructionYear(t *testing.T) {
	operationYear := flexibleInt(1980)
	detail := houseDetailDTO{houseCommonDTO: houseCommonDTO{OperationYear: &operationYear}}
	profile := profileFromDetail(detail, nil, time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC))
	syncLegacyFields(&profile)
	if profile.Characteristics.YearBuilt != nil {
		t.Fatalf("construction year must remain unpublished: %+v", profile.Characteristics)
	}
	if profile.Apartments != nil || profile.YearBuilt == nil || *profile.YearBuilt != 1980 {
		t.Fatalf("legacy operation-year fallback changed: %+v", profile)
	}
}

func TestSquareSummaryFailuresAreOptional(t *testing.T) {
	cases := map[string]func(http.ResponseWriter){
		"status":    func(w http.ResponseWriter) { http.Error(w, "bad", http.StatusBadGateway) },
		"malformed": func(w http.ResponseWriter) { _, _ = w.Write([]byte(`{`)) },
		"oversized": func(w http.ResponseWriter) { _, _ = w.Write([]byte(strings.Repeat("x", maxResponseBytes+1))) },
		"timeout": func(w http.ResponseWriter) {
			time.Sleep(100 * time.Millisecond)
			_, _ = w.Write(arbatSquare)
		},
	}
	for name, squareResponse := range cases {
		t.Run(name, func(t *testing.T) {
			squareRequests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.Contains(r.URL.Path, "searchByFiasHouseCodeList"):
					_, _ = w.Write(arbatLookup)
				case strings.Contains(r.URL.Path, "get-house-square-data"):
					squareRequests++
					squareResponse(w)
				default:
					_, _ = w.Write(arbatDetail)
				}
			}))
			defer server.Close()
			profile, err := NewClient(server.URL, 20*time.Millisecond, server.Client()).FetchHouse(context.Background(), "151b9095-9f1d-4ab5-afc6-db971eed9d49")
			if err != nil {
				t.Fatal(err)
			}
			if squareRequests != 1 || profile.SquareSummaryAvailable || len(profile.SquarePayload) != 0 {
				t.Fatalf("optional source state: requests=%d profile=%+v", squareRequests, profile)
			}
			if len(profile.DataSources) != 2 || profile.DataSources[1].Available {
				t.Fatalf("unexpected sources: %+v", profile.DataSources)
			}
		})
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
