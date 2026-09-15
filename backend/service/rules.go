package service

import (
	"strings"
	"tvoydom/domain"
)

type Classifier interface{ Classify(string) string }
type RuleClassifier struct{}

func (RuleClassifier) Classify(text string) string {
	text = strings.ReplaceAll(strings.ToLower(text), "ё", "е")
	for _, rule := range []struct {
		words    []string
		category string
	}{
		{[]string{"труба", "течет", "протечка"}, "PIPE_LEAK"},
		{[]string{"лифт"}, "ELEVATOR"},
		{[]string{"отопление", "батарея", "холодно"}, "HEATING"},
	} {
		for _, word := range rule.words {
			if strings.Contains(text, word) {
				return rule.category
			}
		}
	}
	return "OTHER"
}

type Router interface {
	Route(string) domain.Organization
}
type RuleRouter struct{}

func (RuleRouter) Route(category string) domain.Organization {
	if category == "ELEVATOR" {
		return domain.Organization{ID: "2", Name: "УК «Тестовая» / подрядчик «ТестЛифт»"}
	}
	return domain.Organization{ID: "1", Name: "УК «Тестовая»"}
}
func NextStatus(status string) (string, error) {
	next := map[string]string{"CREATED": "SENT", "SENT": "ACCEPTED", "ACCEPTED": "IN_PROGRESS", "IN_PROGRESS": "RESOLVED"}[status]
	if next == "" {
		return "", domain.ErrConflict
	}
	return next, nil
}
