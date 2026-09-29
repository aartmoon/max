package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"tvoydom/domain"
	"tvoydom/integration/maxbot"
)

type maxAuthRepoStub struct {
	user        domain.User
	identity    maxbot.Identity
	sessionUser string
	sessionHash string
}

func (r *maxAuthRepoStub) UpsertUserByMAX(_ context.Context, identity maxbot.Identity) (domain.User, error) {
	r.identity = identity
	return r.user, nil
}

func (r *maxAuthRepoStub) CreateSession(_ context.Context, userID, tokenHash string, _ time.Time) error {
	r.sessionUser, r.sessionHash = userID, tokenHash
	return nil
}

func TestMAXAuthCreatesResidentSessionFromValidatedIdentity(t *testing.T) {
	repo := &maxAuthRepoStub{user: domain.User{ID: "42", Name: "Иван", Roles: []string{"resident"}}}
	svc := MAXAuthService{
		Repo: repo, BotToken: "bot-secret", TokenGenerator: func() (string, error) { return "session-token", nil },
		Now: func() time.Time { return time.Unix(100, 0) },
		Validate: func(raw, token string, now time.Time) (maxbot.Identity, error) {
			if raw != "signed" || token != "bot-secret" || !now.Equal(time.Unix(100, 0)) {
				t.Fatal("unexpected validation input")
			}
			return maxbot.Identity{UserID: 77, ChatID: 88, Name: "Иван Петров"}, nil
		},
	}

	session, err := svc.Login(context.Background(), "signed")
	if err != nil {
		t.Fatal(err)
	}
	if session.Token != "session-token" || session.User.ID != "42" || repo.identity.Name != "Иван Петров" || repo.sessionUser != "42" || repo.sessionHash == "" {
		t.Fatalf("unexpected session or persisted data: %+v %+v", session, repo)
	}
}

func TestMAXAuthRequiresEmailForPrivilegedAccounts(t *testing.T) {
	for _, role := range []string{"manager", "admin"} {
		t.Run(role, func(t *testing.T) {
			repo := &maxAuthRepoStub{user: domain.User{ID: "9", Roles: []string{"resident", role}}}
			svc := MAXAuthService{Repo: repo, BotToken: "secret", Validate: func(string, string, time.Time) (maxbot.Identity, error) {
				return maxbot.Identity{UserID: 7, ChatID: 8}, nil
			}}
			if _, err := svc.Login(context.Background(), "signed"); !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("got %v", err)
			}
			if repo.sessionUser != "" {
				t.Fatal("privileged MAX login created a session")
			}
		})
	}
}
