package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"tvoydom/domain"
	"tvoydom/integration/maxbot"
)

func (p Postgres) LinkMAXAccount(ctx context.Context, appUserID string, maxUserID, chatID int64) error {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM user_max_accounts WHERE max_user_id=$1 AND user_id<>$2`, maxUserID, appUserID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO user_max_accounts(user_id,max_user_id,chat_id)
		VALUES($1,$2,$3)
		ON CONFLICT(user_id) DO UPDATE SET max_user_id=EXCLUDED.max_user_id,chat_id=EXCLUDED.chat_id,updated_at=now()`, appUserID, maxUserID, chatID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p Postgres) UpsertUserByMAX(ctx context.Context, identity maxbot.Identity) (domain.User, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, identity.UserID); err != nil {
		return domain.User{}, err
	}
	var userID string
	err = tx.QueryRow(ctx, `SELECT user_id::text FROM user_max_accounts WHERE max_user_id=$1`, identity.UserID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO users(name,updated_at) VALUES($1,now()) RETURNING id::text`, identity.Name).Scan(&userID)
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO user_roles(user_id,role) VALUES($1,'resident')`, userID)
		}
		if err == nil {
			_, err = tx.Exec(ctx, `INSERT INTO user_max_accounts(user_id,max_user_id,chat_id) VALUES($1,$2,$3)`, userID, identity.UserID, identity.ChatID)
		}
	} else if err == nil {
		_, err = tx.Exec(ctx, `UPDATE user_max_accounts SET chat_id=$1,updated_at=now() WHERE max_user_id=$2`, identity.ChatID, identity.UserID)
	}
	if err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	return p.userByID(ctx, userID)
}
