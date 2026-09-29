package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"tvoydom/domain"
	"tvoydom/integration/maxbot"
)

type MAXAuthRepository interface {
	UpsertUserByMAX(context.Context, maxbot.Identity) (domain.User, error)
	CreateSession(context.Context, string, string, time.Time) error
}

type MAXAuthService struct {
	Repo           MAXAuthRepository
	BotToken       string
	Now            func() time.Time
	Validate       MAXLaunchValidator
	TokenGenerator func() (string, error)
	SessionTTL     time.Duration
}

func (s MAXAuthService) Login(ctx context.Context, rawInitData string) (domain.AuthSession, error) {
	if s.Repo == nil || s.BotToken == "" {
		return domain.AuthSession{}, domain.ErrUnauthorized
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	validate := s.Validate
	if validate == nil {
		validate = maxbot.ValidateLaunchData
	}
	identity, err := validate(rawInitData, s.BotToken, now)
	if err != nil {
		return domain.AuthSession{}, domain.ErrUnauthorized
	}
	identity.Name = strings.TrimSpace(identity.Name)
	if identity.Name == "" {
		identity.Name = "Пользователь MAX"
	}
	if utf8.RuneCountInString(identity.Name) > 200 {
		identity.Name = string([]rune(identity.Name)[:200])
	}
	user, err := s.Repo.UpsertUserByMAX(ctx, identity)
	if err != nil {
		return domain.AuthSession{}, err
	}
	if HasRole(user, "manager") || HasRole(user, "admin") {
		return domain.AuthSession{}, domain.ErrForbidden
	}
	auth := AuthService{TokenGenerator: s.TokenGenerator, SessionTTL: s.SessionTTL}
	token, err := auth.token()
	if err != nil {
		return domain.AuthSession{}, err
	}
	if err := s.Repo.CreateSession(ctx, user.ID, hashSecret(token), now.Add(auth.sessionTTL())); err != nil {
		return domain.AuthSession{}, err
	}
	return domain.AuthSession{Token: token, User: user}, nil
}
