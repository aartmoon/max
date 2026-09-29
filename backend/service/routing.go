package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tvoydom/domain"
)

var ErrRoutingRequired = errors.New("для заявки требуется ручная маршрутизация")

type RouteQuery struct {
	HouseID, Category, Place, Urgency string
	At                                time.Time
}

type RouteDecision struct {
	Primary, Contractor, Escalation *domain.ResponsibilityRule
	Reason                          string
}

type ResponsibilityRepository interface {
	FindActiveRules(context.Context, RouteQuery) ([]domain.ResponsibilityRule, error)
}

type RouteResolver interface {
	Resolve(context.Context, RouteQuery) (RouteDecision, error)
}

type RoutingService struct{ Repo ResponsibilityRepository }

func (s RoutingService) Resolve(ctx context.Context, query RouteQuery) (RouteDecision, error) {
	rules, err := s.Repo.FindActiveRules(ctx, query)
	if err != nil {
		return RouteDecision{}, err
	}
	best := map[string][]domain.ResponsibilityRule{}
	scores := map[string]int{}
	for _, rule := range rules {
		if !rule.Active || rule.Category != query.Category || rule.ValidFrom.After(query.At) || (rule.ValidTo != nil && !rule.ValidTo.After(query.At)) {
			continue
		}
		score := 0
		if rule.Place != "" {
			if rule.Place != query.Place {
				continue
			}
			score++
		}
		if rule.Urgency != "" {
			if rule.Urgency != query.Urgency {
				continue
			}
			score++
		}
		if current, ok := scores[rule.Role]; !ok || score > current {
			scores[rule.Role] = score
			best[rule.Role] = []domain.ResponsibilityRule{rule}
		} else if score == current {
			best[rule.Role] = append(best[rule.Role], rule)
		}
	}
	if len(best["PRIMARY"]) != 1 {
		return RouteDecision{}, ErrRoutingRequired
	}
	decision := RouteDecision{Primary: &best["PRIMARY"][0]}
	if len(best["CONTRACTOR"]) == 1 {
		decision.Contractor = &best["CONTRACTOR"][0]
	}
	if len(best["ESCALATION"]) == 1 {
		decision.Escalation = &best["ESCALATION"][0]
	}
	decision.Reason = fmt.Sprintf("Правило дома: категория «%s»", query.Category)
	if query.Place != "" {
		decision.Reason += ", место «" + query.Place + "»"
	}
	if decision.Primary.IsDemo {
		decision.Reason += ". Демонстрационное правило"
	}
	return decision, nil
}

func DetectImmediateDanger(text string) bool {
	normalized := strings.ReplaceAll(strings.ToLower(text), "ё", "е")
	for _, phrase := range []string{"пахнет газ", "запах газ", "утечка газа", "пожар", "горит квартира", "угроза жизни", "угрожает жизни"} {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	return false
}
