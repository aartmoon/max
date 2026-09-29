package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"tvoydom/domain"
)

type SaveOrganizationInput struct {
	Name, Type, INN, OGRN, Phone, Website string
	Active                                bool
}
type SaveRuleInput struct {
	HouseID, OrganizationID, Category, Place, Urgency, Role, Source, SourceURL string
	ValidFrom                                                                  time.Time
	ValidTo                                                                    *time.Time
	Active, IsDemo                                                             bool
}
type RoutingAdminRepository interface {
	SaveOrganization(context.Context, string, SaveOrganizationInput) (domain.Organization, error)
	SaveResponsibilityRule(context.Context, string, SaveRuleInput) (domain.ResponsibilityRule, error)
}
type RoutingAdminService struct{ Repo RoutingAdminRepository }

func (s RoutingAdminService) SaveOrganization(ctx context.Context, id string, in SaveOrganizationInput) (domain.Organization, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return domain.Organization{}, domain.ValidationError{Message: "Название организации обязательно"}
	}
	if utf8.RuneCountInString(in.Name) > 200 {
		return domain.Organization{}, domain.ValidationError{Message: "Название организации не должно превышать 200 символов"}
	}
	if !validOrganizationType(in.Type) {
		return domain.Organization{}, domain.ValidationError{Message: "Неизвестный тип организации"}
	}
	return s.Repo.SaveOrganization(ctx, id, in)
}
func (s RoutingAdminService) SaveRule(ctx context.Context, id string, in SaveRuleInput) (domain.ResponsibilityRule, error) {
	if in.HouseID == "" || in.OrganizationID == "" || strings.TrimSpace(in.Category) == "" {
		return domain.ResponsibilityRule{}, domain.ValidationError{Message: "Дом, организация и категория обязательны"}
	}
	if in.Role != "PRIMARY" && in.Role != "CONTRACTOR" && in.Role != "ESCALATION" {
		return domain.ResponsibilityRule{}, domain.ValidationError{Message: "Неизвестная роль организации"}
	}
	if in.ValidFrom.IsZero() {
		in.ValidFrom = time.Now().UTC()
	}
	in.ValidFrom = in.ValidFrom.UTC()
	if in.ValidTo != nil {
		value := in.ValidTo.UTC()
		in.ValidTo = &value
		if value.Before(in.ValidFrom) {
			return domain.ResponsibilityRule{}, domain.ValidationError{Message: "Дата окончания не может быть раньше даты начала"}
		}
	}
	in.Source = strings.TrimSpace(in.Source)
	if in.IsDemo && in.Source == "" {
		return domain.ResponsibilityRule{}, domain.ValidationError{Message: "Для демонстрационного правила укажите источник"}
	}
	return s.Repo.SaveResponsibilityRule(ctx, id, in)
}
func validOrganizationType(value string) bool {
	switch value {
	case "MANAGEMENT_COMPANY", "HOA_COOPERATIVE", "RESOURCE_SUPPLIER", "CONTRACTOR", "REGIONAL_OPERATOR", "MUNICIPAL", "HOUSING_INSPECTION", "OTHER":
		return true
	}
	return false
}
