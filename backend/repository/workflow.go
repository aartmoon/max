package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"tvoydom/domain"
	"tvoydom/service"
)

func (p Postgres) AssigneeOrganization(ctx context.Context, userID string) (string, error) {
	var id string
	err := p.Pool.QueryRow(ctx, `SELECT uo.organization_id::text FROM user_organizations uo JOIN user_roles ur ON ur.user_id=uo.user_id AND ur.role IN ('manager','admin') WHERE uo.user_id=$1 LIMIT 1`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	return id, err
}

func (p Postgres) UpdateAssignment(ctx context.Context, id string, actor domain.User, in service.AssignmentInput) (domain.Request, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.Request{}, err
	}
	defer tx.Rollback(ctx)
	var status string
	if err = tx.QueryRow(ctx, `SELECT status FROM requests WHERE id=$1 FOR UPDATE`, id).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotFound
	} else if err != nil {
		return domain.Request{}, err
	}
	if status == "CLOSED" || status == "REJECTED" {
		return domain.Request{}, domain.ErrConflict
	}
	_, err = tx.Exec(ctx, `UPDATE requests SET assigned_user_id=NULLIF($1,'')::bigint,contractor_organization_id=COALESCE(NULLIF($2,'')::bigint,contractor_organization_id),visit_start=$3,visit_end=$4,executor_contact=$5 WHERE id=$6`, in.AssigneeUserID, in.ContractorOrganizationID, in.VisitStart, in.VisitEnd, in.ExecutorContact, id)
	if err != nil {
		return domain.Request{}, err
	}
	text := fmt.Sprintf("Назначение обновлено. Исполнитель: %s; визит: %s — %s", in.AssigneeUserID, formatEventTime(in.VisitStart), formatEventTime(in.VisitEnd))
	if _, err = tx.Exec(ctx, `INSERT INTO request_messages(request_id,message_type,message_text,author_user_id,author_name,author_role) VALUES($1,'SYSTEM_EVENT',$2,$3,$4,$5)`, id, text, actor.ID, actor.Name, actorRole(actor)); err != nil {
		return domain.Request{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Request{}, err
	}
	return p.Get(ctx, id, "")
}

func (p Postgres) WorkTransition(ctx context.Context, id string, actor domain.User, to, report string) (domain.Request, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.Request{}, err
	}
	defer tx.Rollback(ctx)
	var from string
	if err = tx.QueryRow(ctx, `SELECT status FROM requests WHERE id=$1 FOR UPDATE`, id).Scan(&from); errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotFound
	} else if err != nil {
		return domain.Request{}, err
	}
	if !repositoryWorkTransition(from, to) {
		return domain.Request{}, domain.ErrConflict
	}
	_, err = tx.Exec(ctx, `UPDATE requests SET status=$1,final_report=CASE WHEN $1='RESOLVED' THEN $2 ELSE final_report END,awaiting_party='NONE' WHERE id=$3`, to, report, id)
	if err != nil {
		return domain.Request{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO request_status_history(request_id,status,comment,actor,actor_user_id,actor_role) VALUES($1,$2,$3,$4,$5,$6)`, id, to, report, actor.Name, actor.ID, actorRole(actor)); err != nil {
		return domain.Request{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO request_messages(request_id,message_type,message_text,author_user_id,author_name,author_role) VALUES($1,'SYSTEM_EVENT',$2,$3,$4,$5)`, id, "Статус изменён: "+to, actor.ID, actor.Name, actorRole(actor)); err != nil {
		return domain.Request{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Request{}, err
	}
	return p.Get(ctx, id, "")
}

func (p Postgres) ResidentDecision(ctx context.Context, id string, actor domain.User, in service.ResidentDecisionInput) (domain.Request, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.Request{}, err
	}
	defer tx.Rollback(ctx)
	var status, owner string
	if err = tx.QueryRow(ctx, `SELECT status,user_id::text FROM requests WHERE id=$1 FOR UPDATE`, id).Scan(&status, &owner); errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotFound
	} else if err != nil {
		return domain.Request{}, err
	}
	if status != "RESOLVED" || owner != actor.ID {
		return domain.Request{}, domain.ErrConflict
	}
	next := "IN_PROGRESS"
	event := "Житель вернул заявку в работу"
	if in.Solved {
		next = "CLOSED"
		event = "Житель подтвердил решение проблемы"
	}
	_, err = tx.Exec(ctx, `UPDATE requests SET status=$1,reopen_count=reopen_count+CASE WHEN $2 THEN 0 ELSE 1 END,awaiting_party='NONE' WHERE id=$3`, next, in.Solved, id)
	if err != nil {
		return domain.Request{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO request_resolution_feedback(request_id,solved,rating,comment,author_user_id) VALUES($1,$2,$3,$4,$5)`, id, in.Solved, in.Rating, in.Comment, actor.ID); err != nil {
		return domain.Request{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO request_status_history(request_id,status,comment,actor,actor_user_id,actor_role) VALUES($1,$2,$3,$4,$5,'resident')`, id, next, in.Comment, actor.Name, actor.ID); err != nil {
		return domain.Request{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO request_messages(request_id,message_type,message_text,author_user_id,author_name,author_role) VALUES($1,'SYSTEM_EVENT',$2,$3,$4,'resident')`, id, event, actor.ID, actor.Name); err != nil {
		return domain.Request{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Request{}, err
	}
	return p.Get(ctx, id, "")
}

func (p Postgres) ManualRoute(ctx context.Context, id string, actor domain.User, primaryID, contractorID, reason string) (domain.Request, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.Request{}, err
	}
	defer tx.Rollback(ctx)
	var status string
	if err = tx.QueryRow(ctx, `SELECT status FROM requests WHERE id=$1 FOR UPDATE`, id).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotFound
	} else if err != nil {
		return domain.Request{}, err
	}
	if status != "ROUTING_REQUIRED" {
		return domain.Request{}, domain.ErrConflict
	}
	var primaryName, contractorName string
	if err = tx.QueryRow(ctx, `SELECT name FROM organizations WHERE id=$1 AND active`, primaryID).Scan(&primaryName); errors.Is(err, pgx.ErrNoRows) {
		return domain.Request{}, domain.ErrNotFound
	} else if err != nil {
		return domain.Request{}, err
	}
	if contractorID != "" {
		if err = tx.QueryRow(ctx, `SELECT name FROM organizations WHERE id=$1 AND active`, contractorID).Scan(&contractorName); err != nil {
			return domain.Request{}, err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE requests SET responsible_organization_id=$1,primary_organization_id=$1,contractor_organization_id=NULLIF($2,'')::bigint,primary_organization_name_snapshot=$3,contractor_organization_name_snapshot=$4,routing_reason=$5,routing_source='manual',status='CREATED' WHERE id=$6`, primaryID, contractorID, primaryName, contractorName, reason, id)
	if err != nil {
		return domain.Request{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO request_status_history(request_id,status,comment,actor,actor_user_id,actor_role) VALUES($1,'CREATED',$2,$3,$4,'admin')`, id, reason, actor.Name, actor.ID); err != nil {
		return domain.Request{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO request_messages(request_id,message_type,message_text,author_user_id,author_name,author_role) VALUES($1,'SYSTEM_EVENT',$2,$3,$4,'admin')`, id, "Заявка направлена: "+primaryName+". "+reason, actor.ID, actor.Name); err != nil {
		return domain.Request{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.Request{}, err
	}
	return p.Get(ctx, id, "")
}

func repositoryWorkTransition(from, to string) bool {
	allowed := map[string][]string{"CREATED": {"ACCEPTED", "REJECTED"}, "SENT": {"ACCEPTED", "REJECTED"}, "ACCEPTED": {"IN_PROGRESS", "REJECTED"}, "IN_PROGRESS": {"RESOLVED", "REJECTED"}}
	for _, candidate := range allowed[from] {
		if candidate == to {
			return true
		}
	}
	return false
}
func actorRole(user domain.User) string {
	if hasRoleValue(user.Roles, "admin") {
		return "admin"
	}
	if hasRoleValue(user.Roles, "manager") {
		return "manager"
	}
	return "resident"
}
func hasRoleValue(roles []string, want string) bool {
	for _, role := range roles {
		if role == want {
			return true
		}
	}
	return false
}
func formatEventTime(value *time.Time) string {
	if value == nil {
		return "не назначено"
	}
	return value.UTC().Format(time.RFC3339)
}
