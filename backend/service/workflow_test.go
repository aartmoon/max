package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"tvoydom/domain"
)

type workflowRepoStub struct {
	accessRepoStub
	assigneeOrganization string
	updated              domain.Request
}

func (r *workflowRepoStub) AssigneeOrganization(context.Context, string) (string, error) {
	return r.assigneeOrganization, nil
}
func (r *workflowRepoStub) UpdateAssignment(_ context.Context, id string, _ domain.User, in AssignmentInput) (domain.Request, error) {
	r.updated = domain.Request{ID: id, AssignedUserID: in.AssigneeUserID, VisitStart: in.VisitStart, VisitEnd: in.VisitEnd}
	return r.updated, nil
}
func (r *workflowRepoStub) WorkTransition(_ context.Context, id string, _ domain.User, status, report string) (domain.Request, error) {
	r.updated = domain.Request{ID: id, Status: status, FinalReport: report}
	return r.updated, nil
}
func (r *workflowRepoStub) ResidentDecision(_ context.Context, id string, _ domain.User, in ResidentDecisionInput) (domain.Request, error) {
	status := "IN_PROGRESS"
	if in.Solved {
		status = "CLOSED"
	}
	r.updated = domain.Request{ID: id, Status: status}
	return r.updated, nil
}
func (r *workflowRepoStub) ManualRoute(context.Context, string, domain.User, string, string, string) (domain.Request, error) {
	return domain.Request{}, nil
}

func TestResolvedRequiresFinalReport(t *testing.T) {
	repo := &workflowRepoStub{accessRepoStub: accessRepoStub{access: domain.RequestAccess{RequestID: "7", PrimaryOrganizationID: "10", Status: "IN_PROGRESS"}}}
	ctx := WithCurrentUser(context.Background(), domain.User{ID: "2", Name: "Менеджер", Roles: []string{"manager"}, OrganizationID: stringPointer("10")})
	_, err := (WorkflowService{Repo: repo, Access: AccessService{Repo: repo}}).SetStatus(ctx, "7", "RESOLVED", ResolveInput{})
	var validation domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("got %v", err)
	}
}

func TestResidentCanCloseOrReopenResolvedRequest(t *testing.T) {
	for _, tc := range []struct {
		solved        bool
		comment, want string
		wantErr       bool
	}{{true, "", "CLOSED", false}, {false, "Не устранено", "IN_PROGRESS", false}, {false, "", "", true}} {
		repo := &workflowRepoStub{accessRepoStub: accessRepoStub{access: domain.RequestAccess{RequestID: "7", OwnerUserID: "1", Status: "RESOLVED"}}}
		ctx := WithCurrentUser(context.Background(), domain.User{ID: "1", Name: "Житель", Roles: []string{"resident"}})
		got, err := (WorkflowService{Repo: repo, Access: AccessService{Repo: repo}}).DecideResolution(ctx, "7", ResidentDecisionInput{Solved: tc.solved, Comment: tc.comment})
		if tc.wantErr && err == nil {
			t.Fatal("empty reopen comment accepted")
		}
		if !tc.wantErr && (err != nil || got.Status != tc.want) {
			t.Fatalf("got %+v %v", got, err)
		}
	}
}

func TestAssignmentRequiresAccessibleOrganizationAndFutureVisit(t *testing.T) {
	now := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	start := now.Add(time.Hour)
	end := start.Add(time.Hour)
	repo := &workflowRepoStub{accessRepoStub: accessRepoStub{access: domain.RequestAccess{RequestID: "7", PrimaryOrganizationID: "10", Status: "IN_PROGRESS"}}, assigneeOrganization: "99"}
	ctx := WithCurrentUser(context.Background(), domain.User{ID: "2", Name: "Менеджер", Roles: []string{"manager"}, OrganizationID: stringPointer("10")})
	svc := WorkflowService{Repo: repo, Access: AccessService{Repo: repo}, Now: func() time.Time { return now }}
	if _, err := svc.UpdateAssignment(ctx, "7", AssignmentInput{AssigneeUserID: "5", VisitStart: &start, VisitEnd: &end}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
	repo.assigneeOrganization = "10"
	past := now.Add(-time.Minute)
	if _, err := svc.UpdateAssignment(ctx, "7", AssignmentInput{AssigneeUserID: "5", VisitStart: &past, VisitEnd: &end}); err == nil {
		t.Fatal("past visit accepted")
	}
}
