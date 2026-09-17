package service

import (
	"context"
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
		r, err := svc.Create(context.Background(), CreateInput{Description: "Тестовая заявка", HouseObjectID: "10", Kind: kind})
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
	calls int
}

func (r *adminCapture) AdminTransition(_ context.Context, id, comment string, next func(string) (string, error)) (domain.Request, error) {
	r.calls++
	status, err := next("CREATED")
	return domain.Request{ID: id, Status: status, UserID: "1"}, err
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
	if repo.calls != 0 {
		t.Fatal("invalid update reached repository")
	}
	if r, err := svc.SetStatus(context.Background(), "1", "REJECTED", "Повторная заявка"); err != nil || r.Status != "REJECTED" {
		t.Fatalf("%+v %v", r, err)
	}
}
