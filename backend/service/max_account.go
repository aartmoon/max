package service

import (
	"context"
	"time"

	"tvoydom/domain"
	"tvoydom/integration/maxbot"
)

type MAXAccountRepository interface {
	LinkMAXAccount(context.Context, string, int64, int64) error
}

type MAXLaunchValidator func(string, string, time.Time) (maxbot.Identity, error)

type MAXAccountService struct {
	Repo     MAXAccountRepository
	Token    string
	Now      func() time.Time
	Validate MAXLaunchValidator
}

func (s MAXAccountService) Link(ctx context.Context, appUserID, rawInitData string) error {
	if appUserID == "" || s.Repo == nil || s.Token == "" {
		return domain.ErrUnauthorized
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	validate := s.Validate
	if validate == nil {
		validate = maxbot.ValidateLaunchData
	}
	identity, err := validate(rawInitData, s.Token, now)
	if err != nil {
		return domain.ErrUnauthorized
	}
	return s.Repo.LinkMAXAccount(ctx, appUserID, identity.UserID, identity.ChatID)
}
