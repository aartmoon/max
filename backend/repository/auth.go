package repository

import (
	"context"
	"errors"
	"strings"
	"time"
	"tvoydom/domain"

	"github.com/jackc/pgx/v5"
)

func (p Postgres) UpsertUserByEmail(ctx context.Context, email string, roles []string) (domain.User, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)
	var user domain.User
	err = tx.QueryRow(ctx, `SELECT id::text,email,name FROM users WHERE lower(email)=lower($1) FOR UPDATE`, email).Scan(&user.ID, &user.Email, &user.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO users(email,name,updated_at) VALUES($1,$1,now()) RETURNING id::text,email,name`, email).Scan(&user.ID, &user.Email, &user.Name)
	}
	if err != nil {
		return domain.User{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET updated_at=now() WHERE id=$1`, user.ID); err != nil {
		return domain.User{}, err
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles(user_id,role) VALUES($1,$2) ON CONFLICT DO NOTHING`, user.ID, role); err != nil {
			return domain.User{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return p.userByID(ctx, user.ID)
}

func (p Postgres) SaveLoginCode(ctx context.Context, email, codeHash string, expiresAt time.Time) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO auth_login_codes(email,code_hash,expires_at) VALUES($1,$2,$3)`, email, codeHash, expiresAt)
	return err
}

func (p Postgres) ConsumeLoginCode(ctx context.Context, email, codeHash string, now time.Time) (domain.User, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)
	var id int64
	err = tx.QueryRow(ctx, `
		SELECT id FROM auth_login_codes
		WHERE lower(email)=lower($1) AND code_hash=$2 AND consumed_at IS NULL AND expires_at>$3
		ORDER BY created_at DESC LIMIT 1
		FOR UPDATE`, email, codeHash, now).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUnauthorized
	}
	if err != nil {
		return domain.User{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE auth_login_codes SET consumed_at=$1 WHERE id=$2`, now, id); err != nil {
		return domain.User{}, err
	}
	var userID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM users WHERE lower(email)=lower($1)`, email).Scan(&userID); err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return p.userByID(ctx, userID)
}

func (p Postgres) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO auth_sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)`, tokenHash, userID, expiresAt)
	return err
}

func (p Postgres) UserBySession(ctx context.Context, tokenHash string, now time.Time) (domain.User, error) {
	var userID string
	err := p.Pool.QueryRow(ctx, `SELECT user_id::text FROM auth_sessions WHERE token_hash=$1 AND expires_at>$2`, tokenHash, now).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUnauthorized
	}
	if err != nil {
		return domain.User{}, err
	}
	return p.userByID(ctx, userID)
}

func (p Postgres) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := p.Pool.Exec(ctx, `DELETE FROM auth_sessions WHERE token_hash=$1`, tokenHash)
	return err
}

func (p Postgres) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT u.id::text,COALESCE(u.email,''),u.name,COALESCE(uo.organization_id::text,'')
		FROM users u LEFT JOIN user_organizations uo ON uo.user_id=u.id
		ORDER BY u.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		var user domain.User
		var orgID string
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &orgID); err != nil {
			return nil, err
		}
		if orgID != "" {
			user.OrganizationID = &orgID
		}
		user.Roles, err = p.userRoles(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (p Postgres) SetUserRoles(ctx context.Context, userID string, roles []string) (domain.User, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id=$1`, userID); err != nil {
		return domain.User{}, err
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles(user_id,role) VALUES($1,$2) ON CONFLICT DO NOTHING`, userID, role); err != nil {
			return domain.User{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return p.userByID(ctx, userID)
}

func (p Postgres) SetUserOrganization(ctx context.Context, userID, organizationID string) (domain.User, error) {
	if strings.TrimSpace(organizationID) == "" {
		_, err := p.Pool.Exec(ctx, `DELETE FROM user_organizations WHERE user_id=$1`, userID)
		if err != nil {
			return domain.User{}, err
		}
		return p.userByID(ctx, userID)
	}
	_, err := p.Pool.Exec(ctx, `INSERT INTO user_organizations(user_id,organization_id) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET organization_id=EXCLUDED.organization_id`, userID, organizationID)
	if err != nil {
		return domain.User{}, err
	}
	return p.userByID(ctx, userID)
}

func (p Postgres) ManagerEmailsForOrganization(ctx context.Context, organizationID string) ([]string, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT u.email FROM users u
		JOIN user_roles ur ON ur.user_id=u.id AND ur.role='manager'
		JOIN user_organizations uo ON uo.user_id=u.id AND uo.organization_id=$1
		WHERE u.email IS NOT NULL AND u.email<>''`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	return emails, rows.Err()
}

func (p Postgres) RequestOwnerEmail(ctx context.Context, requestID string) (string, error) {
	var email string
	err := p.Pool.QueryRow(ctx, `
		SELECT COALESCE(u.email,'')
		FROM requests r JOIN users u ON u.id=r.user_id
		WHERE r.id=$1`, requestID).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	return email, err
}

func (p Postgres) userByID(ctx context.Context, userID string) (domain.User, error) {
	var user domain.User
	var orgID string
	err := p.Pool.QueryRow(ctx, `
		SELECT u.id::text,COALESCE(u.email,''),u.name,COALESCE(uo.organization_id::text,'')
		FROM users u LEFT JOIN user_organizations uo ON uo.user_id=u.id
		WHERE u.id=$1`, userID).Scan(&user.ID, &user.Email, &user.Name, &orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	if orgID != "" {
		user.OrganizationID = &orgID
	}
	user.Roles, err = p.userRoles(ctx, user.ID)
	return user, err
}

func (p Postgres) userRoles(ctx context.Context, userID string) ([]string, error) {
	rows, err := p.Pool.Query(ctx, `SELECT role FROM user_roles WHERE user_id=$1 ORDER BY role`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}
