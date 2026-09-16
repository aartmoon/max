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
}
type AdminService struct {
	Repo          AdminRepository
	Notifications NotificationService
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
	if status == "REJECTED" && comment == "" {
		return domain.Request{}, domain.ValidationError{Message: "Укажите причину отклонения заявки"}
	}
	r, err := s.Repo.AdminTransition(ctx, id, comment, func(from string) (string, error) {
		if err := ValidateAdminTransition(from, status); err != nil {
			return "", err
		}
		return status, nil
	})
	if err == nil {
		s.Notifications.Notify(ctx, r)
	}
	return r, err
}
