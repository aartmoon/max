package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"tvoydom/domain"
	"tvoydom/service"
)

type sessionCookieRepo struct {
	sessions map[string]domain.User
	deleted  []string
}

func (r sessionCookieRepo) UpsertUserByEmail(context.Context, string, []string) (domain.User, error) {
	return domain.User{}, nil
}

func (r sessionCookieRepo) SaveLoginCode(context.Context, string, string, time.Time) error {
	return nil
}

func (r sessionCookieRepo) ConsumeLoginCode(context.Context, string, string, time.Time) (domain.User, error) {
	return domain.User{}, nil
}

func (r sessionCookieRepo) CreateSession(context.Context, string, string, time.Time) error {
	return nil
}

func (r sessionCookieRepo) UserBySession(_ context.Context, tokenHash string, _ time.Time) (domain.User, error) {
	user, ok := r.sessions[tokenHash]
	if !ok {
		return domain.User{}, domain.ErrUnauthorized
	}
	return user, nil
}

func (r *sessionCookieRepo) DeleteSession(_ context.Context, tokenHash string) error {
	r.deleted = append(r.deleted, tokenHash)
	delete(r.sessions, tokenHash)
	return nil
}

func TestCurrentUserAcceptsValidSessionWhenStaleCookieComesFirst(t *testing.T) {
	validToken := "valid-session"
	validHash := hashSessionToken(validToken)
	want := domain.User{ID: "7", Email: "resident@example.test", Roles: []string{"resident"}}
	repo := &sessionCookieRepo{sessions: map[string]domain.User{validHash: want}}
	h := Handler{Auth: service.AuthService{Repo: repo}}
	req := httptest.NewRequest(http.MethodGet, "/api/me/apartments", nil)
	req.Header.Add("Cookie", service.SessionCookieName+"=stale-session")
	req.Header.Add("Cookie", service.SessionCookieName+"="+validToken)

	got, err := h.currentUser(req)

	if err != nil {
		t.Fatalf("expected the valid session cookie to authenticate, got %v", err)
	}
	if got.ID != want.ID {
		t.Fatalf("authenticated the wrong user: got %+v want %+v", got, want)
	}
}

func TestProtectedResponsesAreNotCached(t *testing.T) {
	token := "valid-session"
	user := domain.User{ID: "7", Email: "resident@example.test", Roles: []string{"resident"}}
	repo := &sessionCookieRepo{sessions: map[string]domain.User{hashSessionToken(token): user}}
	h := Handler{Auth: service.AuthService{Repo: repo}}.Routes()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: service.SessionCookieName, Value: token})
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected authenticated /api/me response, status=%d body=%s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("private authenticated response must not be cached, got Cache-Control %q", got)
	}
}

func TestLogoutInvalidatesEverySessionCookieCandidate(t *testing.T) {
	repo := &sessionCookieRepo{sessions: map[string]domain.User{
		hashSessionToken("legacy-session"):  {ID: "7"},
		hashSessionToken("current-session"): {ID: "7"},
	}}
	h := Handler{Auth: service.AuthService{Repo: repo}}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Add("Cookie", service.SessionCookieName+"=legacy-session")
	req.Header.Add("Cookie", service.SessionCookieName+"=current-session")
	res := httptest.NewRecorder()

	h.logout(res, req)

	if len(repo.deleted) != 2 {
		t.Fatalf("logout must invalidate every session candidate, deleted %d", len(repo.deleted))
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("logout left active sessions behind: %+v", repo.sessions)
	}
}

func hashSessionToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
