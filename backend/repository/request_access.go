package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"tvoydom/domain"
)

func (p Postgres) RequestAccess(ctx context.Context, id string) (domain.RequestAccess, error) {
	var a domain.RequestAccess
	err := p.Pool.QueryRow(ctx, `
		SELECT r.id::text,r.user_id::text,COALESCE(r.primary_organization_id::text,''),
		       COALESCE(r.contractor_organization_id::text,''),COALESCE(r.assigned_user_id::text,''),
		       COALESCE(uo.organization_id::text,''),r.status
		FROM requests r LEFT JOIN user_organizations uo ON uo.user_id=r.assigned_user_id
		WHERE r.id=$1`, id).Scan(&a.RequestID, &a.OwnerUserID, &a.PrimaryOrganizationID, &a.ContractorOrganizationID, &a.AssigneeUserID, &a.AssigneeOrganizationID, &a.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, domain.ErrNotFound
	}
	return a, err
}

func (p Postgres) ListAccessibleRequests(ctx context.Context, q domain.AdminRequestQuery) (domain.AdminRequestPage, error) {
	page := domain.AdminRequestPage{Items: []domain.Request{}, Page: q.Page, PageSize: q.PageSize}
	args := []any{q.OrganizationID, q.UserID, q.Now.UTC()}
	where := []string{`($1::text='' OR r.primary_organization_id::text=$1 OR r.contractor_organization_id::text=$1 OR EXISTS(SELECT 1 FROM user_organizations access_uo WHERE access_uo.user_id=r.assigned_user_id AND access_uo.organization_id::text=$1))`, `$2::text IS NOT NULL`, `$3::timestamptz IS NOT NULL`}
	add := func(value any) string { args = append(args, value); return fmt.Sprintf("$%d", len(args)) }

	switch q.Queue {
	case "ACTIVE":
		where = append(where, `r.status NOT IN ('CLOSED','REJECTED')`)
	case "NEW":
		where = append(where, `r.status IN ('ROUTING_REQUIRED','CREATED','SENT')`)
	case "IN_WORK":
		where = append(where, `r.status IN ('ACCEPTED','IN_PROGRESS')`)
	case "UNASSIGNED":
		where = append(where, `r.assigned_user_id IS NULL AND r.status NOT IN ('CLOSED','REJECTED')`)
	case "MINE":
		where = append(where, `r.assigned_user_id::text=$2 AND r.status NOT IN ('CLOSED','REJECTED')`)
	case "VISIT_TODAY":
		start, end := add(q.VisitStart.UTC()), add(q.VisitEnd.UTC())
		where = append(where, fmt.Sprintf(`r.visit_start >= %s AND r.visit_start < %s AND r.status NOT IN ('CLOSED','REJECTED')`, start, end))
	case "OVERDUE":
		where = append(where, `r.deadline < $3 AND r.status NOT IN ('CLOSED','REJECTED')`)
	case "DONE":
		where = append(where, `r.status IN ('CLOSED','REJECTED')`)
	case "EMERGENCY":
		where = append(where, `r.kind='EMERGENCY' AND r.status NOT IN ('CLOSED','REJECTED')`)
	}
	if q.Status != "" {
		where = append(where, "r.status="+add(q.Status))
	}
	if q.Kind != "" {
		where = append(where, "r.kind="+add(q.Kind))
	}
	for _, token := range strings.Fields(strings.ReplaceAll(strings.ToLower(q.Query), "ё", "е")) {
		p := add("%" + token + "%")
		where = append(where, fmt.Sprintf(`replace(lower(concat_ws(' ',r.id::text,COALESCE(NULLIF(r.address_snapshot,''),h.address),r.description,COALESCE(NULLIF(r.primary_organization_name_snapshot,''),po.name,''),COALESCE(NULLIF(r.contractor_organization_name_snapshot,''),co.name,''),COALESCE(au.name,''))),'ё','е') LIKE %s`, p))
	}
	whereSQL := strings.Join(where, " AND ")
	fromSQL := ` FROM requests r JOIN houses h ON h.id=r.house_id LEFT JOIN organizations po ON po.id=COALESCE(r.primary_organization_id,r.responsible_organization_id) LEFT JOIN organizations co ON co.id=r.contractor_organization_id LEFT JOIN users au ON au.id=r.assigned_user_id `
	if err := p.Pool.QueryRow(ctx, `SELECT count(*)`+fromSQL+`WHERE `+whereSQL, args...).Scan(&page.Total); err != nil {
		return page, err
	}

	order := `CASE WHEN r.kind='EMERGENCY' AND r.status NOT IN ('CLOSED','REJECTED') THEN 5 WHEN r.deadline<$3 AND r.status NOT IN ('CLOSED','REJECTED') THEN 4 WHEN r.status IN ('ROUTING_REQUIRED','CREATED','SENT') THEN 3 WHEN r.status='ACCEPTED' THEN 2 WHEN r.status='IN_PROGRESS' THEN 1 ELSE 0 END DESC,r.deadline ASC,r.created_at DESC,r.id DESC`
	if q.Sort == "NEWEST" {
		order = `r.created_at DESC,r.id DESC`
	}
	if q.Sort == "DEADLINE" {
		order = `CASE WHEN r.status IN ('CLOSED','REJECTED') THEN 1 ELSE 0 END,r.deadline ASC,r.created_at DESC,r.id DESC`
	}
	limit, offset := add(q.PageSize), add((q.Page-1)*q.PageSize)
	rows, err := p.Pool.Query(ctx, selectRequest+`WHERE `+whereSQL+` ORDER BY `+order+` LIMIT `+limit+` OFFSET `+offset, args...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return page, err
	}
	summarySQL := `SELECT
	 count(*) FILTER (WHERE r.status NOT IN ('CLOSED','REJECTED')),
	 count(*) FILTER (WHERE r.status IN ('ROUTING_REQUIRED','CREATED','SENT')),
	 count(*) FILTER (WHERE r.deadline<$3 AND r.status NOT IN ('CLOSED','REJECTED')),
	 count(*) FILTER (WHERE r.assigned_user_id IS NULL AND r.status NOT IN ('CLOSED','REJECTED')),
	 count(*) FILTER (WHERE r.status IN ('CLOSED','REJECTED'))` + fromSQL + `WHERE ` + strings.Join(where[:3], " AND ")
	if err := p.Pool.QueryRow(ctx, summarySQL, q.OrganizationID, q.UserID, q.Now.UTC()).Scan(&page.Summary.Active, &page.Summary.New, &page.Summary.Overdue, &page.Summary.Unassigned, &page.Summary.Done); err != nil {
		return page, err
	}
	return page, nil
}
