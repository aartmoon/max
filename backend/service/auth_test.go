package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
	"tvoydom/domain"
)

type authRepo struct {
	users    map[string]domain.User
	codes    map[string]string
	sessions map[string]domain.User
	roles    map[string][]string
}

func newAuthRepo() *authRepo {
	return &authRepo{users: map[string]domain.User{}, codes: map[string]string{}, sessions: map[string]domain.User{}, roles: map[string][]string{}}
}

func (r *authRepo) UpsertUserByEmail(_ context.Context, email string, roles []string) (domain.User, error) {
	u := r.users[email]
	if u.ID == "" {
		u = domain.User{ID: "7", Email: email, Name: email}
	}
	u.Roles = append([]string{}, roles...)
	r.users[email] = u
	r.roles[email] = u.Roles
	return u, nil
}

func (r *authRepo) SaveLoginCode(_ context.Context, email, codeHash string, _ time.Time) error {
	r.codes[email] = codeHash
	return nil
}

func (r *authRepo) ConsumeLoginCode(_ context.Context, email, codeHash string, _ time.Time) (domain.User, error) {
	if r.codes[email] != codeHash {
		return domain.User{}, domain.ErrUnauthorized
	}
	delete(r.codes, email)
	return r.users[email], nil
}

func (r *authRepo) CreateSession(_ context.Context, userID, tokenHash string, _ time.Time) error {
	for _, u := range r.users {
		if u.ID == userID {
			r.sessions[tokenHash] = u
		}
	}
	return nil
}

func (r *authRepo) UserBySession(_ context.Context, tokenHash string, _ time.Time) (domain.User, error) {
	u, ok := r.sessions[tokenHash]
	if !ok {
		return domain.User{}, domain.ErrUnauthorized
	}
	return u, nil
}

func (r *authRepo) DeleteSession(_ context.Context, tokenHash string) error {
	delete(r.sessions, tokenHash)
	return nil
}

type captureMailer struct {
	email string
	code  string
}

func (m *captureMailer) SendLoginCode(_ context.Context, email, code string) error {
	m.email, m.code = email, code
	return nil
}

func (m *captureMailer) SendNewRequest(context.Context, string, domain.Request) error { return nil }

func (m *captureMailer) SendRequestStatusChanged(context.Context, string, domain.Request, string) error {
	return nil
}

func TestAuthRequestCodeCreatesBootstrapAdminAndSendsCode(t *testing.T) {
	repo := newAuthRepo()
	mailer := &captureMailer{}
	svc := AuthService{
		Repo:            repo,
		Mailer:          mailer,
		Now:             func() time.Time { return time.Unix(100, 0).UTC() },
		CodeGenerator:   func() (string, error) { return "123456", nil },
		TokenGenerator:  func() (string, error) { return "token", nil },
		BootstrapAdmins: map[string]bool{"admin@example.com": true},
	}

	if err := svc.RequestCode(context.Background(), " Admin@Example.COM "); err != nil {
		t.Fatal(err)
	}
	if mailer.email != "admin@example.com" || mailer.code != "123456" {
		t.Fatalf("unexpected mail: %+v", mailer)
	}
	if got := repo.roles["admin@example.com"]; len(got) != 2 || got[0] != "resident" || got[1] != "admin" {
		t.Fatalf("admin role was not bootstrapped: %+v", got)
	}
	hash := sha256.Sum256([]byte("admin@example.com:123456"))
	if repo.codes["admin@example.com"] != hex.EncodeToString(hash[:]) {
		t.Fatal("code hash was not stored deterministically")
	}
}

func TestAuthVerifyCodeConsumesCodeAndCreatesSession(t *testing.T) {
	repo := newAuthRepo()
	mailer := &captureMailer{}
	svc := AuthService{
		Repo:           repo,
		Mailer:         mailer,
		Now:            func() time.Time { return time.Unix(100, 0).UTC() },
		CodeGenerator:  func() (string, error) { return "123456", nil },
		TokenGenerator: func() (string, error) { return "session-token", nil },
	}
	if err := svc.RequestCode(context.Background(), "user@example.com"); err != nil {
		t.Fatal(err)
	}

	session, err := svc.VerifyCode(context.Background(), "user@example.com", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if session.Token != "session-token" || session.User.Email != "user@example.com" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if _, exists := repo.codes["user@example.com"]; exists {
		t.Fatal("login code was not consumed")
	}
}
