package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"
	"tvoydom/domain"
)

const SessionCookieName = "td_session"

type contextUserKey struct{}

func WithCurrentUser(ctx context.Context, user domain.User) context.Context {
	return context.WithValue(ctx, contextUserKey{}, user)
}

func CurrentUser(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(contextUserKey{}).(domain.User)
	return user, ok
}

type AuthRepository interface {
	UpsertUserByEmail(ctx context.Context, email string, roles []string) (domain.User, error)
	SaveLoginCode(ctx context.Context, email, codeHash string, expiresAt time.Time) error
	ConsumeLoginCode(ctx context.Context, email, codeHash string, now time.Time) (domain.User, error)
	CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	UserBySession(ctx context.Context, tokenHash string, now time.Time) (domain.User, error)
	DeleteSession(ctx context.Context, tokenHash string) error
}

type Mailer interface {
	SendLoginCode(ctx context.Context, email, code string) error
	SendNewRequest(ctx context.Context, email string, r domain.Request) error
	SendRequestStatusChanged(ctx context.Context, email string, r domain.Request, comment string) error
}

type AuthService struct {
	Repo            AuthRepository
	Mailer          Mailer
	BootstrapAdmins map[string]bool
	Now             func() time.Time
	CodeGenerator   func() (string, error)
	TokenGenerator  func() (string, error)
	CodeTTL         time.Duration
	SessionTTL      time.Duration
}

func (s AuthService) RequestCode(ctx context.Context, rawEmail string) error {
	email, err := normalizeEmail(rawEmail)
	if err != nil {
		return err
	}
	roles := []string{"resident"}
	if s.BootstrapAdmins[email] {
		roles = []string{"resident", "admin"}
	}
	if _, err := s.Repo.UpsertUserByEmail(ctx, email, roles); err != nil {
		return err
	}
	code, err := s.code()
	if err != nil {
		return err
	}
	if err := s.Repo.SaveLoginCode(ctx, email, hashSecret(email+":"+code), s.now().Add(s.codeTTL())); err != nil {
		return err
	}
	return s.Mailer.SendLoginCode(ctx, email, code)
}

func (s AuthService) VerifyCode(ctx context.Context, rawEmail, rawCode string) (domain.AuthSession, error) {
	email, err := normalizeEmail(rawEmail)
	if err != nil {
		return domain.AuthSession{}, err
	}
	code := strings.TrimSpace(rawCode)
	if len(code) != 6 {
		return domain.AuthSession{}, domain.ErrUnauthorized
	}
	user, err := s.Repo.ConsumeLoginCode(ctx, email, hashSecret(email+":"+code), s.now())
	if err != nil {
		return domain.AuthSession{}, err
	}
	token, err := s.token()
	if err != nil {
		return domain.AuthSession{}, err
	}
	if err := s.Repo.CreateSession(ctx, user.ID, hashSecret(token), s.now().Add(s.sessionTTL())); err != nil {
		return domain.AuthSession{}, err
	}
	return domain.AuthSession{Token: token, User: user}, nil
}

func (s AuthService) UserByToken(ctx context.Context, token string) (domain.User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.User{}, domain.ErrUnauthorized
	}
	return s.Repo.UserBySession(ctx, hashSecret(token), s.now())
}

func (s AuthService) Logout(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	return s.Repo.DeleteSession(ctx, hashSecret(token))
}

func (s AuthService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s AuthService) codeTTL() time.Duration {
	if s.CodeTTL > 0 {
		return s.CodeTTL
	}
	return 10 * time.Minute
}

func (s AuthService) sessionTTL() time.Duration {
	if s.SessionTTL > 0 {
		return s.SessionTTL
	}
	return 30 * 24 * time.Hour
}

func (s AuthService) code() (string, error) {
	if s.CodeGenerator != nil {
		return s.CodeGenerator()
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s AuthService) token() (string, error) {
	if s.TokenGenerator != nil {
		return s.TokenGenerator()
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if len(email) > 320 {
		return "", domain.ValidationError{Message: "Некорректная почта"}
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", domain.ValidationError{Message: "Некорректная почта"}
	}
	return email, nil
}

func HasRole(user domain.User, role string) bool {
	for _, current := range user.Roles {
		if current == role {
			return true
		}
	}
	return false
}

func hashSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
