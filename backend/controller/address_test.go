package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tvoydom/domain"
	"tvoydom/service"
)

type addressProviderStub struct {
	search domain.AddressSearch
	items  map[int64]domain.AddressInfo
}

func (s *addressProviderStub) Search(_ context.Context, in domain.AddressSearch) ([]domain.AddressSuggestion, error) {
	s.search = in
	return []domain.AddressSuggestion{{ObjectID: "10", ObjectKind: "house", FullAddress: "г. Москва, д. 1"}}, nil
}

func (s *addressProviderStub) GetAddress(_ context.Context, id int64) (domain.AddressInfo, error) {
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
	provider := &addressProviderStub{items: map[int64]domain.AddressInfo{10: {ObjectID: "10", ObjectKind: "house", FullAddress: "г. Москва, д. 1", IsActive: true}}}
	houses := &houseRepoStub{house: domain.House{ID: "42", GARObjectID: "10", ObjectGUID: "10000000-0000-0000-0000-000000000010", Address: "г. Москва, д. 1"}}
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
}
