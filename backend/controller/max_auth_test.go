package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tvoydom/domain"
	"tvoydom/integration/maxbot"
	"tvoydom/service"
)

type maxLoginRepoStub struct{}

func (maxLoginRepoStub) UpsertUserByMAX(context.Context, maxbot.Identity) (domain.User, error) {
	return domain.User{ID: "42", Name: "Житель MAX", Roles: []string{"resident"}}, nil
}

func (maxLoginRepoStub) CreateSession(context.Context, string, string, time.Time) error {
	return nil
}

func TestMAXAuthCookieWorksInsideHTTPSMiniApp(t *testing.T) {
	handler := Handler{MAXAuth: service.MAXAuthService{
		Repo:     maxLoginRepoStub{},
		BotToken: "bot-secret",
		Validate: func(string, string, time.Time) (maxbot.Identity, error) {
			return maxbot.Identity{UserID: 7, ChatID: 8, Name: "Житель MAX"}, nil
		},
		TokenGenerator: func() (string, error) { return "session-token", nil },
	}}.Routes()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/max", bytes.NewBufferString(`{"initData":"signed"}`))
	req.Header.Set("Content-Type", "application/json")
	req.TLS = &tls.ConnectionState{}
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if !cookies[0].Secure || cookies[0].SameSite != http.SameSiteNoneMode {
		t.Fatalf("cookie Secure = %v, SameSite = %v; want Secure and SameSite=None", cookies[0].Secure, cookies[0].SameSite)
	}
}

func TestMAXAuthIgnoresUntrustedForwardedProtoOnHTTP(t *testing.T) {
	handler := Handler{MAXAuth: service.MAXAuthService{
		Repo:     maxLoginRepoStub{},
		BotToken: "bot-secret",
		Validate: func(string, string, time.Time) (maxbot.Identity, error) {
			return maxbot.Identity{UserID: 7, ChatID: 8, Name: "Житель MAX"}, nil
		},
		TokenGenerator: func() (string, error) { return "session-token", nil },
	}}.Routes()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/max", bytes.NewBufferString(`{"initData":"signed"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie Secure = %v, SameSite = %v; want non-Secure and SameSite=Lax", cookies[0].Secure, cookies[0].SameSite)
	}
}

func TestMAXAuthUsesConfiguredSecureCookieBehindTLSProxy(t *testing.T) {
	handler := Handler{SecureCookies: true, MAXAuth: service.MAXAuthService{
		Repo:     maxLoginRepoStub{},
		BotToken: "bot-secret",
		Validate: func(string, string, time.Time) (maxbot.Identity, error) {
			return maxbot.Identity{UserID: 7, ChatID: 8, Name: "Житель MAX"}, nil
		},
		TokenGenerator: func() (string, error) { return "session-token", nil },
	}}.Routes()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/max", bytes.NewBufferString(`{"initData":"signed"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if !cookies[0].Secure || cookies[0].SameSite != http.SameSiteNoneMode {
		t.Fatalf("cookie Secure = %v, SameSite = %v; want Secure and SameSite=None", cookies[0].Secure, cookies[0].SameSite)
	}
}
