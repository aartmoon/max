package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"tvoydom/integration/maxbot"
)

type maxAccountRepoCapture struct {
	appUserID string
	maxUserID int64
	chatID    int64
}

func (r *maxAccountRepoCapture) LinkMAXAccount(_ context.Context, appUserID string, maxUserID, chatID int64) error {
	r.appUserID, r.maxUserID, r.chatID = appUserID, maxUserID, chatID
	return nil
}

func TestMAXAccountLinkPersistsOnlyValidatedIdentity(t *testing.T) {
	repo := &maxAccountRepoCapture{}
	validator := func(raw, token string, now time.Time) (maxbot.Identity, error) {
		if raw != "signed-data" || token != "secret" || !now.Equal(time.Unix(100, 0)) {
			t.Fatal("validator received unexpected input")
		}
		return maxbot.Identity{UserID: 77, ChatID: 88}, nil
	}
	svc := MAXAccountService{Repo: repo, Token: "secret", Now: func() time.Time { return time.Unix(100, 0) }, Validate: validator}

	if err := svc.Link(context.Background(), "42", "signed-data"); err != nil {
		t.Fatal(err)
	}
	if repo.appUserID != "42" || repo.maxUserID != 77 || repo.chatID != 88 {
		t.Fatalf("unexpected saved link: %+v", repo)
	}
}

func TestMAXAccountLinkRejectsInvalidIdentity(t *testing.T) {
	repo := &maxAccountRepoCapture{}
	svc := MAXAccountService{Repo: repo, Token: "secret", Validate: func(string, string, time.Time) (maxbot.Identity, error) {
		return maxbot.Identity{}, errors.New("invalid")
	}}
	if err := svc.Link(context.Background(), "42", "tampered"); err == nil {
		t.Fatal("invalid identity accepted")
	}
	if repo.appUserID != "" {
		t.Fatal("invalid identity was persisted")
	}
}
