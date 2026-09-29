package repository

import "context"

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
