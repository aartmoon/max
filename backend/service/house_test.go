package service

import (
	"context"
	"errors"
	"testing"
	"tvoydom/domain"
)

type houseAddressProvider struct {
	info *domain.AddressInfo
	err  error
}

func (p houseAddressProvider) Search(context.Context, string, int) ([]domain.AddressSuggestion, error) {
	return nil, nil
}

func (p houseAddressProvider) GetByGUID(context.Context, string) (*domain.AddressInfo, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.info, nil
}

type houseRepository struct {
	byGUID map[string]domain.House
	nextID string
}

func (r *houseRepository) GetHouseByFIASGUID(_ context.Context, guid string) (domain.House, error) {
	h, ok := r.byGUID[guid]
	if !ok {
		return domain.House{}, domain.ErrNotFound
	}
	return h, nil
}

func (r *houseRepository) ResolveHouse(_ context.Context, in domain.HouseIdentity) (domain.House, error) {
	if h, ok := r.byGUID[in.FIASGUID]; ok {
		return h, nil
	}
	h := domain.House{
		ID:              r.nextID,
		FIASGUID:        in.FIASGUID,
		Address:         in.Address,
		CadastralNumber: in.CadastralNumber,
	}
	r.byGUID[in.FIASGUID] = h
	return h, nil
}

func TestResolveHouseCreatesIdentityFromAddressProvider(t *testing.T) {
	cadastral := "16:50:000000:123"
	repo := &houseRepository{byGUID: map[string]domain.House{}, nextID: "42"}
	svc := HouseService{
		Repo: repo,
		Addresses: houseAddressProvider{info: &domain.AddressInfo{
			FIASGUID:        "11111111-2222-3333-4444-555555555555",
			Address:         "Респ Татарстан, г Казань, ул Чистопольская, д 20",
			CadastralNumber: &cadastral,
			ObjectType:      "house",
		}},
	}

	house, err := svc.Resolve(context.Background(), "11111111-2222-3333-4444-555555555555")
	if err != nil {
		t.Fatal(err)
	}
	if house.ID != "42" || house.FIASGUID != "11111111-2222-3333-4444-555555555555" || house.Address == "" {
		t.Fatalf("unexpected house: %+v", house)
	}
	if house.CadastralNumber == nil || *house.CadastralNumber != cadastral {
		t.Fatalf("cadastral number was not persisted: %+v", house.CadastralNumber)
	}
}

func TestResolveHouseReturnsExistingHouseWithoutProviderCall(t *testing.T) {
	existing := domain.House{ID: "7", FIASGUID: "11111111-2222-3333-4444-555555555555", Address: "old address"}
	svc := HouseService{
		Repo:      &houseRepository{byGUID: map[string]domain.House{existing.FIASGUID: existing}},
		Addresses: houseAddressProvider{err: errors.New("provider must not be called")},
	}

	house, err := svc.Resolve(context.Background(), existing.FIASGUID)
	if err != nil {
		t.Fatal(err)
	}
	if house.ID != existing.ID {
		t.Fatalf("expected existing house id %s, got %s", existing.ID, house.ID)
	}
}

func TestResolveHouseRejectsNonHouseAddressObject(t *testing.T) {
	svc := HouseService{
		Repo: &houseRepository{byGUID: map[string]domain.House{}},
		Addresses: houseAddressProvider{info: &domain.AddressInfo{
			FIASGUID:   "11111111-2222-3333-4444-555555555555",
			Address:    "Респ Татарстан, г Казань",
			ObjectType: "city",
		}},
	}

	_, err := svc.Resolve(context.Background(), "11111111-2222-3333-4444-555555555555")
	if !errors.Is(err, domain.ErrInvalidHouse) {
		t.Fatalf("expected invalid house, got %v", err)
	}
}

func TestResolveHouseValidatesGUID(t *testing.T) {
	svc := HouseService{}
	if _, err := svc.Resolve(context.Background(), "not-a-guid"); !errors.Is(err, domain.ErrInvalidFIASGUID) {
		t.Fatalf("expected invalid guid, got %v", err)
	}
}
