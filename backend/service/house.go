package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"tvoydom/domain"
)

type AddressProvider interface {
	Search(context.Context, domain.AddressSearch) ([]domain.AddressSuggestion, error)
	GetAddress(context.Context, int64) (domain.AddressInfo, error)
}

type HouseRepository interface {
	ResolveHouse(context.Context, domain.AddressInfo) (domain.House, error)
}

type HouseProfileRepository interface {
	GetHouseProfile(context.Context, string) (domain.HouseProfile, error)
	SaveHouseProfile(context.Context, domain.HouseProfile) error
	WithHouseProfileLock(context.Context, string, func(context.Context) error) error
}

type GISRegistryClient interface {
	FetchHouse(context.Context, string) (domain.HouseProfile, error)
}

type HouseService struct {
	Repo       HouseRepository
	Addresses  AddressProvider
	Profiles   HouseProfileRepository
	Registry   GISRegistryClient
	ProfileTTL time.Duration
	Now        func() time.Time
}

func (s HouseService) Resolve(ctx context.Context, rawObjectID string) (domain.House, error) {
	objectID, err := parseObjectID(rawObjectID)
	if err != nil {
		return domain.House{}, err
	}
	info, err := s.Addresses.GetAddress(ctx, objectID)
	if err != nil {
		return domain.House{}, err
	}
	if info.ObjectKind != "house" || !info.IsActive {
		return domain.House{}, domain.ErrInvalidHouse
	}
	house, err := s.Repo.ResolveHouse(ctx, info)
	if err != nil {
		return domain.House{}, err
	}
	if s.Profiles == nil || s.Registry == nil {
		return house, nil
	}
	return s.resolveProfile(ctx, house)
}

func (s HouseService) resolveProfile(ctx context.Context, house domain.House) (domain.House, error) {
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	ttl := s.ProfileTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	cached, cacheErr := s.Profiles.GetHouseProfile(ctx, house.ID)
	if cacheErr == nil && cached.FetchedAt.Add(ttl).After(now) {
		return applyProfile(house, cached), nil
	}
	if cacheErr != nil && !errors.Is(cacheErr, domain.ErrNotFound) {
		return domain.House{}, cacheErr
	}
	var result domain.HouseProfile
	err := s.Profiles.WithHouseProfileLock(ctx, house.ID, func(lockCtx context.Context) error {
		current, err := s.Profiles.GetHouseProfile(lockCtx, house.ID)
		if err == nil && current.FetchedAt.Add(ttl).After(now) {
			result = current
			return nil
		}
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		fresh, err := s.Registry.FetchHouse(lockCtx, house.ObjectGUID)
		if err != nil {
			return err
		}
		fresh.HouseID = house.ID
		if fresh.FetchedAt.IsZero() {
			fresh.FetchedAt = now.UTC()
		}
		if err := s.Profiles.SaveHouseProfile(lockCtx, fresh); err != nil {
			return err
		}
		result = fresh
		return nil
	})
	if err != nil {
		if cacheErr == nil {
			cached.Stale = true
			return applyProfile(house, cached), nil
		}
		return domain.House{}, err
	}
	return applyProfile(house, result), nil
}

func applyProfile(house domain.House, profile domain.HouseProfile) domain.House {
	house.CadastralNumber = profile.CadastralNumber
	house.TotalArea = profile.TotalArea
	house.LivingArea = profile.LivingArea
	house.Floors = profile.Floors
	house.Entrances = profile.Entrances
	house.Apartments = profile.Apartments
	house.YearBuilt = profile.YearBuilt
	house.Organization = profile.Organization
	house.Manager = profile.Manager
	house.Contact = profile.Contact
	house.DataSource = "ГИС ЖКХ"
	house.DataUpdatedAt = &profile.FetchedAt
	house.Stale = profile.Stale
	return house
}

func parseObjectID(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, domain.ErrInvalidAddressID
	}
	return value, nil
}
