package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"tvoydom/domain"
	"tvoydom/integration"
)

func TestNewRequestKinds(t *testing.T) {
	for _, kind := range []string{"EMERGENCY", "COMPLAINT"} {
		repo := &captureRepository{}
		addresses := fakeAddressProvider{items: map[int64]domain.AddressInfo{10: {ObjectID: "10", ObjectKind: "house", FullAddress: "Тестовый дом 1", IsActive: true}}}
		svc := requestServiceForTest(repo, addresses)
		r, err := svc.Create(authContext(), CreateInput{Description: "Тестовая заявка", HouseObjectID: "10", Kind: kind})
		if err != nil || r.Kind != kind {
			t.Fatalf("%s: %+v %v", kind, r, err)
		}
	}
}
func TestAdminTransitions(t *testing.T) {
	for _, pair := range [][2]string{{"CREATED", "ACCEPTED"}, {"SENT", "ACCEPTED"}, {"ACCEPTED", "IN_PROGRESS"}, {"IN_PROGRESS", "RESOLVED"}, {"CREATED", "REJECTED"}} {
		if err := ValidateAdminTransition(pair[0], pair[1]); err != nil {
			t.Errorf("%v: %v", pair, err)
		}
	}
	for _, pair := range [][2]string{{"CREATED", "RESOLVED"}, {"RESOLVED", "IN_PROGRESS"}, {"REJECTED", "ACCEPTED"}, {"ACCEPTED", "ACCEPTED"}, {"IN_PROGRESS", "CREATED"}, {"CREATED", "invalid"}} {
		if err := ValidateAdminTransition(pair[0], pair[1]); err == nil {
			t.Errorf("accepted %v", pair)
		}
	}
}

type adminCapture struct {
	AdminRepository
	calls       int
	from        string
	lastComment string
	ownerEmail  string
}

func (r *adminCapture) AdminTransition(_ context.Context, id, comment string, next func(string) (string, error)) (domain.Request, error) {
	r.calls++
	r.lastComment = comment
	from := r.from
	if from == "" {
		from = "CREATED"
	}
	status, err := next(from)
	return domain.Request{ID: id, Status: status, UserID: "1"}, err
}

func (r *adminCapture) RequestOwnerEmail(_ context.Context, id string) (string, error) {
	if r.ownerEmail == "" {
		return "resident@example.com", nil
	}
	return r.ownerEmail, nil
}

type statusMailerCapture struct {
	email   string
	request domain.Request
	comment string
	err     error
}

func (m *statusMailerCapture) SendLoginCode(context.Context, string, string) error {
	return nil
}

func (m *statusMailerCapture) SendNewRequest(context.Context, string, domain.Request) error {
	return nil
}

func (m *statusMailerCapture) SendRequestStatusChanged(_ context.Context, email string, r domain.Request, comment string) error {
	m.email = email
	m.request = r
	m.comment = comment
	return m.err
}

func TestAdminCommentValidation(t *testing.T) {
	repo := &adminCapture{}
	svc := AdminService{Repo: repo, Notifications: NotificationService{Client: integration.MockMaxClient{}}}
	if _, err := svc.SetStatus(context.Background(), "1", "REJECTED", "  "); err == nil {
		t.Fatal("empty rejection reason accepted")
	}
	if _, err := svc.SetStatus(context.Background(), "1", "ACCEPTED", strings.Repeat("я", 2001)); err == nil {
		t.Fatal("long comment accepted")
	}
	repo.from = "IN_PROGRESS"
	if _, err := svc.SetStatus(context.Background(), "1", "RESOLVED", "  "); err == nil {
		t.Fatal("empty resolution result accepted")
	}
	if repo.calls != 0 {
		t.Fatal("invalid update reached repository")
	}
	repo.from = "CREATED"
	if r, err := svc.SetStatus(context.Background(), "1", "REJECTED", "Повторная заявка"); err != nil || r.Status != "REJECTED" {
		t.Fatalf("%+v %v", r, err)
	}
	if repo.lastComment != "Повторная заявка" {
		t.Fatalf("comment was not normalized: %q", repo.lastComment)
	}
}

func TestAdminStatusChangeEmailsRequestOwner(t *testing.T) {
	repo := &adminCapture{}
	mailer := &statusMailerCapture{}
	svc := AdminService{Repo: repo, Notifications: NotificationService{Client: integration.MockMaxClient{}}, Mailer: mailer}

	r, err := svc.SetStatus(context.Background(), "1", "ACCEPTED", "  Приняли в работу  ")
	if err != nil {
		t.Fatal(err)
	}
	if mailer.email != "resident@example.com" || mailer.request.ID != r.ID || mailer.request.Status != "ACCEPTED" {
		t.Fatalf("status email was not sent to request owner: %+v", mailer)
	}
	if mailer.comment != "Приняли в работу" {
		t.Fatalf("comment was not included: %q", mailer.comment)
	}
}

func TestAdminStatusChangeIgnoresEmailFailure(t *testing.T) {
	repo := &adminCapture{}
	mailer := &statusMailerCapture{err: errors.New("smtp unavailable")}
	svc := AdminService{Repo: repo, Notifications: NotificationService{Client: integration.MockMaxClient{}}, Mailer: mailer}

	r, err := svc.SetStatus(context.Background(), "1", "ACCEPTED", "")
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "ACCEPTED" || mailer.email != "resident@example.com" {
		t.Fatalf("unexpected result after mail failure: request=%+v mailer=%+v", r, mailer)
	}
}
