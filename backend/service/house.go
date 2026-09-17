package service

import (
	"context"
	"strconv"
	"strings"

	"tvoydom/domain"
)

type AddressProvider interface {
	Search(context.Context, domain.AddressSearch) ([]domain.AddressSuggestion, error)
	GetAddress(context.Context, int64) (domain.AddressInfo, error)
}

type HouseRepository interface {
	ResolveHouse(context.Context, domain.AddressInfo) (domain.House, error)
}

type HouseService struct {
	Repo      HouseRepository
	Addresses AddressProvider
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
	return s.Repo.ResolveHouse(ctx, info)
}

func parseObjectID(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, domain.ErrInvalidAddressID
	}
	return value, nil
}
