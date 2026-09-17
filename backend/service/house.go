package service

import (
	"context"
	"regexp"
	"strings"
	"tvoydom/domain"
)

var guidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type AddressProvider interface {
	Search(ctx context.Context, query string, limit int) ([]domain.AddressSuggestion, error)
	GetByGUID(ctx context.Context, fiasGUID string) (*domain.AddressInfo, error)
}

type HouseRepository interface {
	GetHouseByFIASGUID(ctx context.Context, fiasGUID string) (domain.House, error)
	ResolveHouse(ctx context.Context, in domain.HouseIdentity) (domain.House, error)
}

type HouseService struct {
	Repo      HouseRepository
	Addresses AddressProvider
}

func (s HouseService) Resolve(ctx context.Context, fiasGUID string) (domain.House, error) {
	fiasGUID = strings.ToLower(strings.TrimSpace(fiasGUID))
	if !guidPattern.MatchString(fiasGUID) {
		return domain.House{}, domain.ErrInvalidFIASGUID
	}
	if h, err := s.Repo.GetHouseByFIASGUID(ctx, fiasGUID); err == nil {
		return h, nil
	} else if err != domain.ErrNotFound {
		return domain.House{}, err
	}
	info, err := s.Addresses.GetByGUID(ctx, fiasGUID)
	if err != nil {
		return domain.House{}, err
	}
	if info == nil || info.ObjectType != "house" {
		return domain.House{}, domain.ErrInvalidHouse
	}
	return s.Repo.ResolveHouse(ctx, domain.HouseIdentity{
		FIASGUID:        fiasGUID,
		Address:         info.Address,
		CadastralNumber: info.CadastralNumber,
	})
}
