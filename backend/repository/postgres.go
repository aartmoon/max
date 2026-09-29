package repository

import (
	"context"
	_ "embed"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tvoydom/domain"
)

//go:embed schema.sql
var schema string

type Postgres struct{ Pool *pgxpool.Pool }

//go:embed migrations/002_admin_house.sql
var adminHouseMigration string

//go:embed migrations/003_gar_house_identity.sql
var garHouseIdentityMigration string

//go:embed migrations/004_gar_address_selection.sql
var garAddressSelectionMigration string

//go:embed migrations/005_gis_house_profiles.sql
var gisHouseProfilesMigration string

//go:embed migrations/006_gar_search_dedup.sql
var garSearchDedupMigration string

//go:embed migrations/007_enriched_gis_house_profiles.sql
var enrichedGISHouseProfilesMigration string

//go:embed migrations/008_auth_roles.sql
var authRolesMigration string

//go:embed migrations/009_request_workflow.sql
var requestWorkflowMigration string

//go:embed migrations/010_arbat_responsibility_snapshot.sql
var arbatResponsibilitySnapshotMigration string

//go:embed migrations/011_arbat_capital_repair.sql
var arbatCapitalRepairMigration string

//go:embed migrations/012_max_accounts.sql
var maxAccountsMigration string

func (p Postgres) Migrate(ctx context.Context) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(740016)`); err != nil {
		return err
	}
	var baseSchemaExists bool
	if err = tx.QueryRow(ctx, `SELECT to_regclass('requests') IS NOT NULL`).Scan(&baseSchemaExists); err != nil {
		return err
	}
	if !baseSchemaExists {
		if _, err = tx.Exec(ctx, schema); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version integer PRIMARY KEY)`); err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=2)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, adminHouseMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(2)`); err != nil {
			return err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=3)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, garHouseIdentityMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(3)`); err != nil {
			return err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=4)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, garAddressSelectionMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(4)`); err != nil {
			return err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=5)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, gisHouseProfilesMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(5)`); err != nil {
			return err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=6)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, garSearchDedupMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(6)`); err != nil {
			return err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=7)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, enrichedGISHouseProfilesMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(7)`); err != nil {
			return err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=8)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, authRolesMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(8)`); err != nil {
			return err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=9)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, requestWorkflowMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(9)`); err != nil {
			return err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=10)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, arbatResponsibilitySnapshotMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(10)`); err != nil {
			return err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=11)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, arbatCapitalRepairMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(11)`); err != nil {
			return err
		}
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=12)`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, maxAccountsMigration); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations VALUES(12)`); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

const selectRequest = `SELECT r.id::text,r.user_id::text,r.house_id::text,r.description,r.problem_type,COALESCE(r.responsible_organization_id::text,''),r.status,r.deadline,r.created_at,r.kind,COALESCE(NULLIF(r.address_snapshot,''),h.address),COALESCE(r.house_object_id::text,''),COALESCE(r.house_object_guid::text,''),COALESCE(r.apartment_object_id::text,''),COALESCE(r.apartment_object_guid::text,''),COALESCE(NULLIF(r.primary_organization_name_snapshot,''),po.name,''),r.request_text,COALESCE(octet_length(r.photo),0)>0,r.problem_place,r.urgency,COALESCE(r.primary_organization_id::text,''),COALESCE(NULLIF(r.primary_organization_name_snapshot,''),po.name,''),COALESCE(r.contractor_organization_id::text,''),COALESCE(NULLIF(r.contractor_organization_name_snapshot,''),co.name,''),COALESCE(r.routing_rule_id::text,''),r.routing_reason,r.routing_source,COALESCE(r.assigned_user_id::text,''),COALESCE(au.name,''),r.visit_start,r.visit_end,r.executor_contact,r.final_report,r.awaiting_party,r.reopen_count FROM requests r JOIN houses h ON h.id=r.house_id LEFT JOIN organizations po ON po.id=COALESCE(r.primary_organization_id,r.responsible_organization_id) LEFT JOIN organizations co ON co.id=r.contractor_organization_id LEFT JOIN users au ON au.id=r.assigned_user_id `

func scan(row pgx.Row) (r domain.Request, err error) {
	err = row.Scan(&r.ID, &r.UserID, &r.HouseID, &r.Description, &r.ProblemType, &r.ResponsibleOrganizationID, &r.Status, &r.Deadline, &r.CreatedAt, &r.Kind, &r.Address, &r.HouseObjectID, &r.HouseObjectGUID, &r.ApartmentObjectID, &r.ApartmentObjectGUID, &r.ResponsibleOrganization, &r.Text, &r.HasPhoto, &r.ProblemPlace, &r.Urgency, &r.PrimaryOrganizationID, &r.PrimaryOrganization, &r.ContractorOrganizationID, &r.ContractorOrganization, &r.RoutingRuleID, &r.RoutingReason, &r.RoutingSource, &r.AssignedUserID, &r.AssignedUserName, &r.VisitStart, &r.VisitEnd, &r.ExecutorContact, &r.FinalReport, &r.AwaitingParty, &r.ReopenCount)
	if errors.Is(err, pgx.ErrNoRows) {
		err = domain.ErrNotFound
	}
	return
}
func (p Postgres) Create(ctx context.Context, r domain.Request) (domain.Request, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return r, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `INSERT INTO requests(user_id,house_id,description,problem_type,responsible_organization_id,status,deadline,created_at,kind,request_text,photo,photo_type,address_snapshot,house_object_id,house_object_guid,apartment_object_id,apartment_object_guid,problem_place,urgency,primary_organization_id,contractor_organization_id,routing_rule_id,primary_organization_name_snapshot,contractor_organization_name_snapshot,routing_reason,routing_source,awaiting_party) VALUES($1,$2,$3,$4,NULLIF($5,'')::bigint,$6,$7,$8,$9,$10,$11,$12,$13,NULLIF($14,'')::bigint,NULLIF($15,'')::uuid,NULLIF($16,'')::bigint,NULLIF($17,'')::uuid,$18,$19,NULLIF($20,'')::bigint,NULLIF($21,'')::bigint,NULLIF($22,'')::bigint,$23,$24,$25,$26,$27) RETURNING id::text`, r.UserID, r.HouseID, r.Description, r.ProblemType, r.ResponsibleOrganizationID, r.Status, r.Deadline, r.CreatedAt, r.Kind, r.Text, r.Photo, r.PhotoType, r.Address, r.HouseObjectID, r.HouseObjectGUID, r.ApartmentObjectID, r.ApartmentObjectGUID, r.ProblemPlace, r.Urgency, r.PrimaryOrganizationID, r.ContractorOrganizationID, r.RoutingRuleID, r.PrimaryOrganization, r.ContractorOrganization, r.RoutingReason, r.RoutingSource, r.AwaitingParty).Scan(&r.ID)
	if err != nil {
		return r, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO request_status_history(request_id,status,created_at,actor,actor_user_id,actor_role) SELECT $1,$2,$3,u.name,u.id,'resident' FROM users u WHERE u.id=$4`, r.ID, r.Status, r.CreatedAt, r.UserID)
	if err != nil {
		return r, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO request_messages(request_id,message_type,message_text,author_user_id,author_name,author_role,created_at) SELECT $1,'SYSTEM_EVENT',$2,u.id,u.name,'resident',$3 FROM users u WHERE u.id=$4`, r.ID, r.RoutingReason, r.CreatedAt, r.UserID)
	if err != nil {
		return r, err
	}
	return r, tx.Commit(ctx)
}
func (p Postgres) List(ctx context.Context, user string) ([]domain.Request, error) {
	rows, err := p.Pool.Query(ctx, selectRequest+`WHERE ($1::text='' OR r.user_id::text=$1) ORDER BY r.created_at DESC,r.id DESC`, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Request{}
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
func (p Postgres) ListByOrganization(ctx context.Context, organizationID string) ([]domain.Request, error) {
	rows, err := p.Pool.Query(ctx, selectRequest+`WHERE r.primary_organization_id::text=$1 OR r.contractor_organization_id::text=$1 OR EXISTS(SELECT 1 FROM user_organizations uo WHERE uo.user_id=r.assigned_user_id AND uo.organization_id::text=$1) ORDER BY r.created_at DESC,r.id DESC`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Request{}
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
func (p Postgres) Get(ctx context.Context, id, user string) (domain.Request, error) {
	return scan(p.Pool.QueryRow(ctx, selectRequest+`WHERE r.id=$1 AND ($2::text='' OR r.user_id::text=$2)`, id, user))
}
func (p Postgres) History(ctx context.Context, id, user string) ([]domain.RequestStatusHistory, error) {
	if _, err := p.Get(ctx, id, user); err != nil {
		return nil, err
	}
	rows, err := p.Pool.Query(ctx, `SELECT id,request_id::text,status,created_at,comment,actor FROM request_status_history WHERE request_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.RequestStatusHistory{}
	for rows.Next() {
		var h domain.RequestStatusHistory
		if err := rows.Scan(&h.ID, &h.RequestID, &h.Status, &h.CreatedAt, &h.Comment, &h.Actor); err != nil {
			return nil, err
		}
		result = append(result, h)
	}
	return result, rows.Err()
}
func (p Postgres) Transition(ctx context.Context, id, user string, next func(string) (string, error)) (domain.Request, error) {
	return p.transition(ctx, id, user, "", "Демо", next)
}
func (p Postgres) AdminTransition(ctx context.Context, id, comment string, next func(string) (string, error)) (domain.Request, error) {
	return p.transition(ctx, id, "", comment, "УК · демо", next)
}
func (p Postgres) transition(ctx context.Context, id, user, comment, actor string, next func(string) (string, error)) (domain.Request, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.Request{}, err
	}
	defer tx.Rollback(ctx)
	r, err := scan(tx.QueryRow(ctx, selectRequest+`WHERE r.id=$1 AND ($2::text='' OR r.user_id::text=$2) FOR UPDATE OF r`, id, user))
	if err != nil {
		return r, err
	}
	r.Status, err = next(r.Status)
	if err != nil {
		return r, err
	}
	if _, err = tx.Exec(ctx, `UPDATE requests SET status=$1 WHERE id=$2`, r.Status, id); err != nil {
		return r, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO request_status_history(request_id,status,comment,actor) VALUES($1,$2,$3,$4)`, id, r.Status, comment, actor); err != nil {
		return r, err
	}
	return r, tx.Commit(ctx)
}
func (p Postgres) Photo(ctx context.Context, id, user string) ([]byte, string, error) {
	var b []byte
	var mime string
	err := p.Pool.QueryRow(ctx, `SELECT photo,photo_type FROM requests WHERE id=$1 AND ($2::text='' OR user_id::text=$2)`, id, user).Scan(&b, &mime)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && len(b) == 0) {
		return nil, "", domain.ErrNotFound
	}
	return b, mime, err
}
