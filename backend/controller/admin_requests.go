package controller

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"tvoydom/domain"
	"tvoydom/service"
)

var adminQueues = map[string]bool{"ACTIVE": true, "NEW": true, "IN_WORK": true, "UNASSIGNED": true, "MINE": true, "VISIT_TODAY": true, "OVERDUE": true, "DONE": true, "EMERGENCY": true, "ALL": true}
var adminSorts = map[string]bool{"PRIORITY": true, "NEWEST": true, "DEADLINE": true}
var adminStatuses = map[string]bool{"ROUTING_REQUIRED": true, "CREATED": true, "SENT": true, "ACCEPTED": true, "IN_PROGRESS": true, "RESOLVED": true, "CLOSED": true, "REJECTED": true}
var adminKinds = map[string]bool{"PROBLEM": true, "APPLICATION": true, "QUESTION": true, "EMERGENCY": true, "COMPLAINT": true}

func parseAdminRequestQuery(values url.Values, user domain.User, now time.Time) (domain.AdminRequestQuery, error) {
	q := domain.AdminRequestQuery{UserID: user.ID, Queue: "ACTIVE", Sort: "PRIORITY", Page: 1, PageSize: 20, Now: now.UTC()}
	if raw := values.Get("page"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			return q, domain.ValidationError{Message: "Некорректная страница"}
		}
		q.Page = v
	}
	if raw := values.Get("pageSize"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || (v != 20 && v != 50 && v != 100) {
			return q, domain.ValidationError{Message: "Некорректный размер страницы"}
		}
		q.PageSize = v
	}
	if raw := values.Get("queue"); raw != "" {
		q.Queue = strings.ToUpper(raw)
	}
	if !adminQueues[q.Queue] {
		return q, domain.ValidationError{Message: "Неизвестная очередь"}
	}
	if raw := values.Get("sort"); raw != "" {
		q.Sort = strings.ToUpper(raw)
	}
	if !adminSorts[q.Sort] {
		return q, domain.ValidationError{Message: "Неизвестная сортировка"}
	}
	q.Status, q.Kind, q.Query = strings.ToUpper(values.Get("status")), strings.ToUpper(values.Get("kind")), strings.TrimSpace(values.Get("q"))
	if q.Status != "" && !adminStatuses[q.Status] {
		return q, domain.ValidationError{Message: "Неизвестный статус"}
	}
	if q.Kind != "" && !adminKinds[q.Kind] {
		return q, domain.ValidationError{Message: "Неизвестный тип заявки"}
	}
	if service.HasRole(user, "admin") {
		q.OrganizationID = strings.TrimSpace(values.Get("organization"))
	} else if user.OrganizationID != nil {
		q.OrganizationID = *user.OrganizationID
	} else {
		return q, domain.ErrForbidden
	}
	if q.Queue == "VISIT_TODAY" {
		start, e1 := time.Parse(time.RFC3339, values.Get("visitStart"))
		end, e2 := time.Parse(time.RFC3339, values.Get("visitEnd"))
		if e1 != nil || e2 != nil || !start.Before(end) {
			return q, domain.ValidationError{Message: "Некорректный интервал визита"}
		}
		q.VisitStart, q.VisitEnd = start.UTC(), end.UTC()
	}
	return q, nil
}
