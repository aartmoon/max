package service

import (
	"context"
	"errors"
	"testing"

	"tvoydom/domain"
)

type messageRepoStub struct {
	accessRepoStub
	saved       domain.RequestMessage
	attachments []NewAttachment
	awaiting    string
}

func (r *messageRepoStub) ListMessages(context.Context, string, bool) ([]domain.RequestMessage, error) {
	return nil, nil
}
func (r *messageRepoStub) CreateMessage(_ context.Context, m domain.RequestMessage, a []NewAttachment, awaiting string) (domain.RequestMessage, error) {
	r.saved = m
	r.attachments = a
	r.awaiting = awaiting
	m.ID = "8"
	return m, nil
}
func (r *messageRepoStub) MessageAttachment(context.Context, string) (domain.MessageAttachmentContent, error) {
	return domain.MessageAttachmentContent{}, domain.ErrNotFound
}

func TestResidentCannotCreateInternalNote(t *testing.T) {
	repo := &messageRepoStub{accessRepoStub: accessRepoStub{access: domain.RequestAccess{RequestID: "7", OwnerUserID: "1", Status: "IN_PROGRESS"}}}
	ctx := WithCurrentUser(context.Background(), domain.User{ID: "1", Name: "Житель", Roles: []string{"resident"}})
	_, err := (MessageService{Repo: repo, Access: AccessService{Repo: repo}}).Create(ctx, NewMessageInput{RequestID: "7", Type: "INTERNAL_NOTE", Text: "Скрыть"})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestResidentReplyDoesNotChangeStatusAndReturnsToOrganization(t *testing.T) {
	repo := &messageRepoStub{accessRepoStub: accessRepoStub{access: domain.RequestAccess{RequestID: "7", OwnerUserID: "1", Status: "IN_PROGRESS"}}}
	ctx := WithCurrentUser(context.Background(), domain.User{ID: "1", Name: "Житель", Roles: []string{"resident"}})
	got, err := (MessageService{Repo: repo, Access: AccessService{Repo: repo}}).Create(ctx, NewMessageInput{RequestID: "7", Type: "RESIDENT_PUBLIC", Text: "Уточнение"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "RESIDENT_PUBLIC" || repo.awaiting != "ORGANIZATION" || repo.access.Status != "IN_PROGRESS" {
		t.Fatalf("message=%+v awaiting=%s", got, repo.awaiting)
	}
}

func TestMessageRejectsSpoofedImage(t *testing.T) {
	repo := &messageRepoStub{accessRepoStub: accessRepoStub{access: domain.RequestAccess{RequestID: "7", OwnerUserID: "1", Status: "IN_PROGRESS"}}}
	ctx := WithCurrentUser(context.Background(), domain.User{ID: "1", Name: "Житель", Roles: []string{"resident"}})
	_, err := (MessageService{Repo: repo, Access: AccessService{Repo: repo}}).Create(ctx, NewMessageInput{RequestID: "7", Type: "RESIDENT_PUBLIC", Attachments: []NewAttachment{{Name: "photo.jpg", Data: []byte("not an image")}}})
	var validation domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("got %v", err)
	}
}
