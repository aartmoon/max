package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tvoydom/domain"
	"tvoydom/service"
)

type addressProviderStub struct {
	search domain.AddressSearch
	items  map[int64]domain.AddressInfo
	err    error
}

func (s *addressProviderStub) Search(_ context.Context, in domain.AddressSearch) ([]domain.AddressSuggestion, error) {
	s.search = in
	return []domain.AddressSuggestion{{ObjectID: "10", ObjectKind: "house", FullAddress: "г. Москва, д. 1"}}, nil
}

func (s *addressProviderStub) GetAddress(_ context.Context, id int64) (domain.AddressInfo, error) {
	if s.err != nil {
		return domain.AddressInfo{}, s.err
	}
	item, ok := s.items[id]
	if !ok {
		return domain.AddressInfo{}, domain.ErrNotFound
	}
	return item, nil
}

type houseRepoStub struct{ house domain.House }

func (s *houseRepoStub) GetHouseByGARObjectID(context.Context, int64) (domain.House, error) {
	return domain.House{}, domain.ErrNotFound
}

func (s *houseRepoStub) ResolveHouse(context.Context, domain.AddressInfo) (domain.House, error) {
	return s.house, nil
}

func TestSearchAddressesValidatesAndPassesFilters(t *testing.T) {
	provider := &addressProviderStub{}
	h := Handler{Addresses: provider}.Routes()
	req := httptest.NewRequest(http.MethodGet, "/api/addresses/search?q=%D0%A2%D0%B2%D0%B5%D1%80%D1%81%D0%BA%D0%B0%D1%8F&kind=house&parentObjectId=100&limit=5", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if provider.search.Query != "Тверская" || provider.search.Kind != "house" || provider.search.ParentObjectID == nil || *provider.search.ParentObjectID != 100 || provider.search.Limit != 5 {
		t.Fatalf("unexpected search: %+v", provider.search)
	}
}

func TestSearchAddressesRejectsInvalidRequests(t *testing.T) {
	h := Handler{Addresses: &addressProviderStub{}}.Routes()
	for _, target := range []string{
		"/api/addresses/search",
		"/api/addresses/search?q=x&kind=planet",
		"/api/addresses/search?parentObjectId=nope&kind=apartment",
		"/api/addresses/search?q=x&limit=51",
	} {
		t.Run(target, func(t *testing.T) {
			res := httptest.NewRecorder()
			h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, target, nil))
			if res.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
		})
	}
}

func TestAddressDetailUsesNumericGARObjectID(t *testing.T) {
	provider := &addressProviderStub{items: map[int64]domain.AddressInfo{10: {ObjectID: "10", ObjectKind: "house", IsActive: true}}}
	h := Handler{Addresses: provider}.Routes()
	for target, wantStatus := range map[string]int{"/api/addresses/10": 200, "/api/addresses/not-a-number": 400} {
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, target, nil))
		if res.Code != wantStatus {
			t.Fatalf("%s: status=%d body=%s", target, res.Code, res.Body.String())
		}
	}
}

func TestResolveHouseAcceptsObjectID(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	floors := 16
	premises := 15
	ogrn := "5147746267906"
	organization := `ГБУ "Жилищник района Арбат"`
	provider := &addressProviderStub{items: map[int64]domain.AddressInfo{10: {ObjectID: "10", ObjectKind: "house", FullAddress: "г. Москва, д. 1", IsActive: true}}}
	houses := &houseRepoStub{house: domain.House{
		ID: "42", GARObjectID: "10", ObjectGUID: "10000000-0000-0000-0000-000000000010", Address: "г. Москва, д. 1",
		Floors: &floors, Apartments: &premises, Organization: &organization, DataSource: "ГИС ЖКХ", DataUpdatedAt: &now, Stale: true,
		Characteristics: domain.HouseCharacteristics{Floors: &floors, ResidentialPremises: &premises},
		Management:      domain.HouseManagement{ShortName: &organization, OGRN: &ogrn},
		DataSources:     []domain.HouseDataSource{{Name: "ГИС ЖКХ · карточка дома", Available: true, UpdatedAt: now, Stale: true}},
	}}
	h := Handler{Addresses: provider, Houses: service.HouseService{Addresses: provider, Repo: houses}}.Routes()
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/houses/resolve", bytes.NewBufferString(`{"objectId":"10"}`)))
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["garObjectId"] != "10" || body["objectGuid"] == "" {
		t.Fatalf("unexpected body: %v", body)
	}
	if body["floors"] != float64(16) || body["dataSource"] != "ГИС ЖКХ" || body["dataUpdatedAt"] == nil || body["stale"] != true {
		t.Fatalf("profile fields missing: %v", body)
	}
	characteristics, ok := body["characteristics"].(map[string]any)
	if !ok || characteristics["residentialPremises"] != float64(15) {
		t.Fatalf("structured characteristics missing: %v", body)
	}
	management, ok := body["management"].(map[string]any)
	if !ok || management["ogrn"] != ogrn {
		t.Fatalf("structured management missing: %v", body)
	}
	sources, ok := body["dataSources"].([]any)
	if !ok || len(sources) != 1 {
		t.Fatalf("sources missing: %v", body)
	}
	if body["apartments"] != float64(15) || body["organization"] != organization {
		t.Fatalf("legacy fields changed: %v", body)
	}
	if _, exposed := body["rawPayload"]; exposed {
		t.Fatal("raw payload must stay private")
	}
}

func TestResolveHouseMapsProfileErrors(t *testing.T) {
	for want, code := range map[error]int{domain.ErrHouseProfileNotFound: http.StatusNotFound, domain.ErrHouseProfileUnavailable: http.StatusServiceUnavailable} {
		provider := &addressProviderStub{err: want}
		h := Handler{Addresses: provider, Houses: service.HouseService{Addresses: provider, Repo: &houseRepoStub{}}}.Routes()
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/houses/resolve", bytes.NewBufferString(`{"objectId":"10"}`)))
		if res.Code != code {
			t.Fatalf("error=%v status=%d body=%s", want, res.Code, res.Body.String())
		}
	}
}

func TestLegacyHouseRouteIsRemoved(t *testing.T) {
	res := httptest.NewRecorder()
	Handler{}.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/house", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
