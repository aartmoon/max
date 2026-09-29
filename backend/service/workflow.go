package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"tvoydom/domain"
)

type AssignmentInput struct {
	AssigneeUserID, ContractorOrganizationID, ExecutorContact string
	VisitStart, VisitEnd                                      *time.Time
}
type ResolveInput struct{ FinalReport string }
type ResidentDecisionInput struct {
	Solved  bool
	Rating  *int
	Comment string
}

type WorkflowRepository interface {
	AccessRepository
	AssigneeOrganization(context.Context, string) (string, error)
	UpdateAssignment(context.Context, string, domain.User, AssignmentInput) (domain.Request, error)
	WorkTransition(context.Context, string, domain.User, string, string) (domain.Request, error)
	ResidentDecision(context.Context, string, domain.User, ResidentDecisionInput) (domain.Request, error)
	ManualRoute(context.Context, string, domain.User, string, string, string) (domain.Request, error)
}
type WorkflowService struct {
	Repo   WorkflowRepository
	Access AccessService
	Now    func() time.Time
}

func (s WorkflowService) UpdateAssignment(ctx context.Context, id string, in AssignmentInput) (domain.Request, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return domain.Request{}, domain.ErrUnauthorized
	}
	access, err := s.Access.Require(ctx, id, CapabilityManage)
	if err != nil {
		return domain.Request{}, err
	}
	if access.Status == "CLOSED" || access.Status == "REJECTED" {
		return domain.Request{}, domain.ErrConflict
	}
	if (in.VisitStart == nil) != (in.VisitEnd == nil) {
		return domain.Request{}, domain.ValidationError{Message: "Укажите начало и окончание визита"}
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	if in.VisitStart != nil {
		start, end := in.VisitStart.UTC(), in.VisitEnd.UTC()
		in.VisitStart, in.VisitEnd = &start, &end
		if start.After(end) {
			return domain.Request{}, domain.ValidationError{Message: "Начало визита не может быть позже окончания"}
		}
		if start.Before(now) {
			return domain.Request{}, domain.ValidationError{Message: "Нельзя назначить визит в прошлом"}
		}
	}
	if in.AssigneeUserID != "" {
		organizationID, err := s.Repo.AssigneeOrganization(ctx, in.AssigneeUserID)
		if err != nil {
			return domain.Request{}, err
		}
		if organizationID != access.PrimaryOrganizationID && organizationID != access.ContractorOrganizationID {
			return domain.Request{}, domain.ErrForbidden
		}
	}
	if in.ContractorOrganizationID != "" && in.ContractorOrganizationID != access.ContractorOrganizationID && in.ContractorOrganizationID != access.PrimaryOrganizationID {
		return domain.Request{}, domain.ErrForbidden
	}
	in.ExecutorContact = strings.TrimSpace(in.ExecutorContact)
	if utf8.RuneCountInString(in.ExecutorContact) > 300 {
		return domain.Request{}, domain.ValidationError{Message: "Контакт исполнителя не должен превышать 300 символов"}
	}
	return s.Repo.UpdateAssignment(ctx, id, user, in)
}

func (s WorkflowService) SetStatus(ctx context.Context, id, status string, in ResolveInput) (domain.Request, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return domain.Request{}, domain.ErrUnauthorized
	}
	access, err := s.Access.Require(ctx, id, CapabilityManage)
	if err != nil {
		return domain.Request{}, err
	}
	report := strings.TrimSpace(in.FinalReport)
	if status == "RESOLVED" && report == "" {
		return domain.Request{}, domain.ValidationError{Message: "Опишите результат выполненных работ"}
	}
	if utf8.RuneCountInString(report) > 5000 {
		return domain.Request{}, domain.ValidationError{Message: "Итоговый отчёт не должен превышать 5000 символов"}
	}
	if !validWorkTransition(access.Status, status) {
		return domain.Request{}, domain.ErrConflict
	}
	return s.Repo.WorkTransition(ctx, id, user, status, report)
}

func validWorkTransition(from, to string) bool {
	allowed := map[string][]string{"CREATED": {"ACCEPTED", "REJECTED"}, "SENT": {"ACCEPTED", "REJECTED"}, "ACCEPTED": {"IN_PROGRESS", "REJECTED"}, "IN_PROGRESS": {"RESOLVED", "REJECTED"}}
	for _, candidate := range allowed[from] {
		if candidate == to {
			return true
		}
	}
	return false
}

func (s WorkflowService) DecideResolution(ctx context.Context, id string, in ResidentDecisionInput) (domain.Request, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return domain.Request{}, domain.ErrUnauthorized
	}
	access, err := s.Access.Require(ctx, id, CapabilityPublic)
	if err != nil {
		return domain.Request{}, err
	}
	if user.ID != access.OwnerUserID {
		return domain.Request{}, domain.ErrForbidden
	}
	if access.Status != "RESOLVED" {
		return domain.Request{}, domain.ErrConflict
	}
	in.Comment = strings.TrimSpace(in.Comment)
	if !in.Solved && in.Comment == "" {
		return domain.Request{}, domain.ValidationError{Message: "Опишите, почему проблема не решена"}
	}
	if in.Rating != nil && (*in.Rating < 1 || *in.Rating > 5) {
		return domain.Request{}, domain.ValidationError{Message: "Оценка должна быть от 1 до 5"}
	}
	if utf8.RuneCountInString(in.Comment) > 2000 {
		return domain.Request{}, domain.ValidationError{Message: "Комментарий не должен превышать 2000 символов"}
	}
	return s.Repo.ResidentDecision(ctx, id, user, in)
}

func (s WorkflowService) RouteManually(ctx context.Context, id, primaryID, contractorID, reason string) (domain.Request, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return domain.Request{}, domain.ErrUnauthorized
	}
	if !HasRole(user, "admin") {
		return domain.Request{}, domain.ErrForbidden
	}
	if strings.TrimSpace(primaryID) == "" {
		return domain.Request{}, domain.ValidationError{Message: "Выберите основную ответственную организацию"}
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "Ручное назначение администратором"
	}
	return s.Repo.ManualRoute(ctx, id, user, primaryID, contractorID, reason)
}
