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

func (p Postgres) Migrate(ctx context.Context) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(740016)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, schema); err != nil {
		return err
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
	return tx.Commit(ctx)
}

const selectRequest = `SELECT r.id::text,r.user_id::text,r.house_id::text,r.description,r.problem_type,r.responsible_organization_id::text,r.status,r.deadline,r.created_at,r.kind,h.address,o.name,r.request_text,COALESCE(octet_length(r.photo),0)>0 FROM requests r JOIN houses h ON h.id=r.house_id JOIN organizations o ON o.id=r.responsible_organization_id `

func scan(row pgx.Row) (r domain.Request, err error) {
	err = row.Scan(&r.ID, &r.UserID, &r.HouseID, &r.Description, &r.ProblemType, &r.ResponsibleOrganizationID, &r.Status, &r.Deadline, &r.CreatedAt, &r.Kind, &r.Address, &r.ResponsibleOrganization, &r.Text, &r.HasPhoto)
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
	err = tx.QueryRow(ctx, `INSERT INTO houses(address) VALUES($1) ON CONFLICT(address) DO UPDATE SET address=EXCLUDED.address RETURNING id::text`, r.Address).Scan(&r.HouseID)
	if err != nil {
		return r, err
	}
	err = tx.QueryRow(ctx, `INSERT INTO requests(user_id,house_id,description,problem_type,responsible_organization_id,status,deadline,created_at,kind,request_text,photo,photo_type) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id::text`, r.UserID, r.HouseID, r.Description, r.ProblemType, r.ResponsibleOrganizationID, r.Status, r.Deadline, r.CreatedAt, r.Kind, r.Text, r.Photo, r.PhotoType).Scan(&r.ID)
	if err != nil {
		return r, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO request_status_history(request_id,status,created_at) VALUES($1,$2,$3)`, r.ID, r.Status, r.CreatedAt)
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
