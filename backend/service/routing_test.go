package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"tvoydom/domain"
)

type ruleRepoStub struct{ rules []domain.ResponsibilityRule }

func (r ruleRepoStub) FindActiveRules(context.Context, RouteQuery) ([]domain.ResponsibilityRule, error) {
	return r.rules, nil
}

func TestRoutingKeepsContractorSeparateAndPrefersSpecificRule(t *testing.T) {
	now := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	rules := []domain.ResponsibilityRule{
		{ID: "1", Role: "PRIMARY", Category: "ELEVATOR", OrganizationID: "10", OrganizationName: "Общая УО", Active: true},
		{ID: "2", Role: "PRIMARY", Category: "ELEVATOR", Place: "COMMON_PROPERTY", OrganizationID: "11", OrganizationName: "УО дома", Active: true},
		{ID: "3", Role: "CONTRACTOR", Category: "ELEVATOR", OrganizationID: "20", OrganizationName: "Лифтовая компания", Active: true},
	}
	got, err := (RoutingService{Repo: ruleRepoStub{rules: rules}}).Resolve(context.Background(), RouteQuery{Category: "ELEVATOR", Place: "COMMON_PROPERTY", At: now})
	if err != nil {
		t.Fatal(err)
	}
	if got.Primary == nil || got.Primary.OrganizationID != "11" || got.Contractor == nil || got.Contractor.OrganizationID != "20" {
		t.Fatalf("unexpected decision: %+v", got)
	}
}

func TestRoutingRequiresManualAssignmentForMissingOrAmbiguousPrimary(t *testing.T) {
	for _, rules := range [][]domain.ResponsibilityRule{
		nil,
		{{ID: "1", Role: "PRIMARY", Category: "HEATING", OrganizationID: "10", Active: true}, {ID: "2", Role: "PRIMARY", Category: "HEATING", OrganizationID: "11", Active: true}},
	} {
		_, err := (RoutingService{Repo: ruleRepoStub{rules: rules}}).Resolve(context.Background(), RouteQuery{Category: "HEATING", At: time.Now().UTC()})
		if !errors.Is(err, ErrRoutingRequired) {
			t.Fatalf("expected manual routing, got %v", err)
		}
	}
}

func TestDetectImmediateDanger(t *testing.T) {
	for _, text := range []string{"Пахнет газом в квартире", "В подъезде пожар", "Есть непосредственная угроза жизни"} {
		if !DetectImmediateDanger(text) {
			t.Fatalf("danger was not detected: %q", text)
		}
	}
	if DetectImmediateDanger("Не работает лифт") {
		t.Fatal("ordinary request marked dangerous")
	}
}
