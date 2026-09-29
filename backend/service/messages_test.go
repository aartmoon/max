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
	internal    bool
}

func (r *messageRepoStub) ListMessages(_ context.Context, _ string, internal bool) ([]domain.RequestMessage, error) {
	r.internal = internal
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

func TestResidentPortalReplyWorksForUserWhoIsAlsoStaff(t *testing.T) {
	repo := &messageRepoStub{accessRepoStub: accessRepoStub{access: domain.RequestAccess{RequestID: "7", OwnerUserID: "1", Status: "IN_PROGRESS"}}}
	ctx := WithCurrentUser(context.Background(), domain.User{ID: "1", Name: "Администратор-житель", Roles: []string{"resident", "admin"}})
	got, err := (MessageService{Repo: repo, Access: AccessService{Repo: repo}}).Create(ctx, NewMessageInput{RequestID: "7", Type: "RESIDENT_PUBLIC", Text: "Добрый день"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "RESIDENT_PUBLIC" || got.AuthorRole != "resident" || repo.awaiting != "ORGANIZATION" {
		t.Fatalf("message=%+v awaiting=%s", got, repo.awaiting)
	}
}

func TestStaffCannotWriteAsResidentWhenTheyDoNotOwnTheRequest(t *testing.T) {
	for _, roles := range [][]string{{"admin"}, {"manager"}} {
		t.Run(roles[0], func(t *testing.T) {
			organizationID := "10"
			repo := &messageRepoStub{accessRepoStub: accessRepoStub{access: domain.RequestAccess{RequestID: "7", OwnerUserID: "1", PrimaryOrganizationID: organizationID, Status: "IN_PROGRESS"}}}
			ctx := WithCurrentUser(context.Background(), domain.User{ID: "2", Name: "Сотрудник", Roles: roles, OrganizationID: &organizationID})
			_, err := (MessageService{Repo: repo, Access: AccessService{Repo: repo}}).Create(ctx, NewMessageInput{RequestID: "7", Type: "RESIDENT_PUBLIC", Text: "От имени жителя"})
			if !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestResidentPortalHidesInternalNotesFromDualRoleUser(t *testing.T) {
	repo := &messageRepoStub{accessRepoStub: accessRepoStub{access: domain.RequestAccess{RequestID: "7", OwnerUserID: "1", Status: "IN_PROGRESS"}}}
	ctx := WithCurrentUser(context.Background(), domain.User{ID: "1", Name: "Администратор-житель", Roles: []string{"resident", "admin"}})
	if _, err := (MessageService{Repo: repo, Access: AccessService{Repo: repo}}).List(ctx, "7"); err != nil {
		t.Fatal(err)
	}
	if repo.internal {
		t.Fatal("resident portal included internal notes")
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
