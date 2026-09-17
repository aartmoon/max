package service

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"tvoydom/domain"
)

type fakeAddressProvider struct{ items map[int64]domain.AddressInfo }

func (p fakeAddressProvider) Search(context.Context, domain.AddressSearch) ([]domain.AddressSuggestion, error) {
	return []domain.AddressSuggestion{}, nil
}

type fakeProfiles struct {
	profile domain.HouseProfile
	err     error
	saves   int
}

func (p *fakeProfiles) GetHouseProfile(context.Context, string) (domain.HouseProfile, error) {
	return p.profile, p.err
}
func (p *fakeProfiles) SaveHouseProfile(_ context.Context, profile domain.HouseProfile) error {
	p.profile, p.err, p.saves = profile, nil, p.saves+1
	return nil
}
func (p *fakeProfiles) WithHouseProfileLock(ctx context.Context, _ string, fn func(context.Context) error) error {
	return fn(ctx)
}

type fakeRegistry struct {
	profile domain.HouseProfile
	err     error
	calls   int
}

func (r *fakeRegistry) FetchHouse(context.Context, string) (domain.HouseProfile, error) {
	r.calls++
	return r.profile, r.err
}

func profileService(now time.Time, profiles *fakeProfiles, registry *fakeRegistry) HouseService {
	return HouseService{
		Repo: &fakeHouseRepository{byObjectID: map[string]domain.House{}, nextID: "42"},
		Addresses: fakeAddressProvider{items: map[int64]domain.AddressInfo{10: {
			ObjectID: "10", ObjectGUID: "11111111-2222-3333-4444-555555555555", ObjectKind: "house", FullAddress: "Москва, Тверская, 1", IsActive: true,
		}}},
		Profiles: profiles, Registry: registry, ProfileTTL: 24 * time.Hour, Now: func() time.Time { return now },
	}
}

func TestHouseProfileFreshCacheSkipsRegistry(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	profiles := &fakeProfiles{profile: domain.HouseProfile{HouseID: "42", GISHouseGUID: "gis", FetchedAt: now.Add(-time.Hour)}}
	registry := &fakeRegistry{}
	house, err := profileService(now, profiles, registry).Resolve(context.Background(), "10")
	if err != nil || registry.calls != 0 || house.DataSource != "ГИС ЖКХ" || house.DataUpdatedAt == nil || house.Stale {
		t.Fatalf("house=%+v calls=%d err=%v", house, registry.calls, err)
	}
}

func TestHouseProfileMissingCacheFetchesAndSaves(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	profiles := &fakeProfiles{err: domain.ErrNotFound}
	registry := &fakeRegistry{profile: domain.HouseProfile{GISHouseGUID: "gis", FetchedAt: now}}
	house, err := profileService(now, profiles, registry).Resolve(context.Background(), "10")
	if err != nil || registry.calls != 1 || profiles.saves != 1 || profiles.profile.HouseID != "42" || house.DataUpdatedAt == nil {
		t.Fatalf("house=%+v profile=%+v calls=%d saves=%d err=%v", house, profiles.profile, registry.calls, profiles.saves, err)
	}
}

func TestHouseProfileStaleFallback(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	fetched := now.Add(-48 * time.Hour)
	profiles := &fakeProfiles{profile: domain.HouseProfile{HouseID: "42", GISHouseGUID: "old", FetchedAt: fetched}}
	registry := &fakeRegistry{err: domain.ErrHouseProfileUnavailable}
	house, err := profileService(now, profiles, registry).Resolve(context.Background(), "10")
	if err != nil || !house.Stale || house.DataUpdatedAt == nil || !house.DataUpdatedAt.Equal(fetched) {
		t.Fatalf("house=%+v err=%v", house, err)
	}
}

func TestHouseProfileMissingPropagatesRegistryErrors(t *testing.T) {
	for _, want := range []error{domain.ErrHouseProfileNotFound, domain.ErrHouseProfileUnavailable} {
		profiles := &fakeProfiles{err: domain.ErrNotFound}
		registry := &fakeRegistry{err: want}
		_, err := profileService(time.Now(), profiles, registry).Resolve(context.Background(), "10")
		if !errors.Is(err, want) {
			t.Fatalf("want %v, got %v", want, err)
		}
	}
}

func (p fakeAddressProvider) GetAddress(_ context.Context, objectID int64) (domain.AddressInfo, error) {
	item, ok := p.items[objectID]
	if !ok {
		return domain.AddressInfo{}, domain.ErrNotFound
	}
	return item, nil
}

type fakeHouseRepository struct {
	byObjectID map[string]domain.House
	nextID     string
}

func (r *fakeHouseRepository) GetHouseByGARObjectID(_ context.Context, objectID int64) (domain.House, error) {
	h, ok := r.byObjectID[strconv.FormatInt(objectID, 10)]
	if !ok {
		return domain.House{}, domain.ErrNotFound
	}
	return h, nil
}

func (r *fakeHouseRepository) ResolveHouse(_ context.Context, info domain.AddressInfo) (domain.House, error) {
	if h, ok := r.byObjectID[info.ObjectID]; ok {
		return h, nil
	}
	h := domain.House{ID: r.nextID, GARObjectID: info.ObjectID, ObjectGUID: info.ObjectGUID, Address: info.FullAddress}
	r.byObjectID[info.ObjectID] = h
	return h, nil
}

func TestResolveHouseCreatesIdentityFromGARObject(t *testing.T) {
	repo := &fakeHouseRepository{byObjectID: map[string]domain.House{}, nextID: "42"}
	svc := HouseService{Repo: repo, Addresses: fakeAddressProvider{items: map[int64]domain.AddressInfo{
		10: {ObjectID: "10", ObjectGUID: "11111111-2222-3333-4444-555555555555", ObjectKind: "house", FullAddress: "г. Москва, ул. Тверская, д. 1", IsActive: true},
	}}}
	house, err := svc.Resolve(context.Background(), "10")
	if err != nil {
		t.Fatal(err)
	}
	if house.ID != "42" || house.GARObjectID != "10" || house.ObjectGUID == "" || house.Address == "" {
		t.Fatalf("unexpected house: %+v", house)
	}
}

func TestResolveHouseReturnsExistingHouseAfterValidatingCurrentGARObject(t *testing.T) {
	existing := domain.House{ID: "7", GARObjectID: "10", Address: "old address"}
	svc := HouseService{
		Repo: &fakeHouseRepository{byObjectID: map[string]domain.House{"10": existing}},
		Addresses: fakeAddressProvider{items: map[int64]domain.AddressInfo{
			10: {ObjectID: "10", ObjectKind: "house", FullAddress: "new address", IsActive: true},
		}},
	}
	house, err := svc.Resolve(context.Background(), "10")
	if err != nil || house.ID != "7" {
		t.Fatalf("house=%+v err=%v", house, err)
	}
}

func TestResolveHouseRejectsInactiveAndNonHouseObjects(t *testing.T) {
	for name, info := range map[string]domain.AddressInfo{
		"wrong kind": {ObjectID: "10", ObjectKind: "apartment", IsActive: true},
		"inactive":   {ObjectID: "10", ObjectKind: "house", IsActive: false},
	} {
		t.Run(name, func(t *testing.T) {
			svc := HouseService{Repo: &fakeHouseRepository{byObjectID: map[string]domain.House{}}, Addresses: fakeAddressProvider{items: map[int64]domain.AddressInfo{10: info}}}
			_, err := svc.Resolve(context.Background(), "10")
			if !errors.Is(err, domain.ErrInvalidHouse) {
				t.Fatalf("expected invalid house, got %v", err)
			}
		})
	}
}

func TestResolveHouseValidatesObjectID(t *testing.T) {
	svc := HouseService{}
	for _, value := range []string{"", "not-a-number", "0", "-1"} {
		if _, err := svc.Resolve(context.Background(), value); !errors.Is(err, domain.ErrInvalidAddressID) {
			t.Fatalf("%q: expected invalid id, got %v", value, err)
		}
	}
}
