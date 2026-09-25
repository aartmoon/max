package service

import (
	"context"
	"strings"
	"tvoydom/domain"
	"unicode/utf8"
)

type AdminRepository interface {
	RequestRepository
	AdminTransition(context.Context, string, string, func(string) (string, error)) (domain.Request, error)
	CreateOrganization(context.Context, string) (domain.Organization, error)
}
type AdminService struct {
	Repo          AdminRepository
	Notifications NotificationService
	Mailer        Mailer
}

func ValidateAdminTransition(from, to string) error {
	allowed := map[string][]string{"CREATED": {"ACCEPTED", "REJECTED"}, "SENT": {"ACCEPTED", "REJECTED"}, "ACCEPTED": {"IN_PROGRESS", "REJECTED"}, "IN_PROGRESS": {"RESOLVED", "REJECTED"}}
	for _, status := range allowed[from] {
		if status == to {
			return nil
		}
	}
	return domain.ErrConflict
}
func (s AdminService) SetStatus(ctx context.Context, id, status, comment string) (domain.Request, error) {
	comment = strings.TrimSpace(comment)
	if utf8.RuneCountInString(comment) > 2000 {
		return domain.Request{}, domain.ValidationError{Message: "Комментарий не должен превышать 2000 символов"}
	}
	if (status == "REJECTED" || status == "RESOLVED") && comment == "" {
		message := "Укажите причину отклонения заявки"
		if status == "RESOLVED" {
			message = "Опишите результат выполненных работ"
		}
		return domain.Request{}, domain.ValidationError{Message: message}
	}
	r, err := s.Repo.AdminTransition(ctx, id, comment, func(from string) (string, error) {
		if err := ValidateAdminTransition(from, status); err != nil {
			return "", err
		}
		return status, nil
	})
	if err == nil {
		s.Notifications.Notify(ctx, r)
		notifyRequestOwnerStatusChanged(ctx, s.Repo, s.Mailer, r, comment)
	}
	return r, err
}

func (s AdminService) CreateOrganization(ctx context.Context, name string) (domain.Organization, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Organization{}, domain.ValidationError{Message: "Название УК обязательно"}
	}
	if utf8.RuneCountInString(name) > 200 {
		return domain.Organization{}, domain.ValidationError{Message: "Название УК не должно превышать 200 символов"}
	}
	return s.Repo.CreateOrganization(ctx, name)
}
