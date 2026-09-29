package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"tvoydom/domain"
	"tvoydom/service"
)

func (p Postgres) ListMessages(ctx context.Context, requestID string, includeInternal bool) ([]domain.RequestMessage, error) {
	rows, err := p.Pool.Query(ctx, `
		SELECT id::text,request_id::text,message_type,message_text,COALESCE(author_user_id::text,''),author_name,author_role,created_at
		FROM request_messages WHERE request_id=$1 AND ($2 OR message_type<>'INTERNAL_NOTE') ORDER BY id`, requestID, includeInternal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.RequestMessage{}
	for rows.Next() {
		var item domain.RequestMessage
		if err := rows.Scan(&item.ID, &item.RequestID, &item.Type, &item.Text, &item.AuthorID, &item.AuthorName, &item.AuthorRole, &item.CreatedAt); err != nil {
			return nil, err
		}
		attachments, err := p.messageAttachments(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		item.Attachments = attachments
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p Postgres) messageAttachments(ctx context.Context, messageID string) ([]domain.MessageAttachment, error) {
	rows, err := p.Pool.Query(ctx, `SELECT id::text,message_id::text,original_name,mime_type,byte_size,created_at FROM request_message_attachments WHERE message_id=$1 ORDER BY id`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.MessageAttachment{}
	for rows.Next() {
		var a domain.MessageAttachment
		if err := rows.Scan(&a.ID, &a.MessageID, &a.Name, &a.MIMEType, &a.Size, &a.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (p Postgres) CreateMessage(ctx context.Context, m domain.RequestMessage, attachments []service.NewAttachment, awaiting string) (domain.RequestMessage, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return m, err
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()
	err = tx.QueryRow(ctx, `INSERT INTO request_messages(request_id,message_type,message_text,author_user_id,author_name,author_role,created_at) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id::text`, m.RequestID, m.Type, m.Text, m.AuthorID, m.AuthorName, m.AuthorRole, now).Scan(&m.ID)
	if err != nil {
		return m, err
	}
	m.CreatedAt = now
	m.Attachments = []domain.MessageAttachment{}
	for _, input := range attachments {
		var a domain.MessageAttachment
		err = tx.QueryRow(ctx, `INSERT INTO request_message_attachments(message_id,original_name,mime_type,byte_size,content,created_at) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text,message_id::text,original_name,mime_type,byte_size,created_at`, m.ID, input.Name, input.MIMEType, len(input.Data), input.Data, now).Scan(&a.ID, &a.MessageID, &a.Name, &a.MIMEType, &a.Size, &a.CreatedAt)
		if err != nil {
			return m, err
		}
		m.Attachments = append(m.Attachments, a)
	}
	if awaiting != "NONE" {
		if _, err = tx.Exec(ctx, `UPDATE requests SET awaiting_party=$1 WHERE id=$2`, awaiting, m.RequestID); err != nil {
			return m, err
		}
	}
	return m, tx.Commit(ctx)
}

func (p Postgres) MessageAttachment(ctx context.Context, id string) (domain.MessageAttachmentContent, error) {
	var a domain.MessageAttachmentContent
	err := p.Pool.QueryRow(ctx, `SELECT a.id::text,a.message_id::text,a.original_name,a.mime_type,a.byte_size,a.created_at,m.request_id::text,a.content FROM request_message_attachments a JOIN request_messages m ON m.id=a.message_id WHERE a.id=$1`, id).Scan(&a.ID, &a.MessageID, &a.Name, &a.MIMEType, &a.Size, &a.CreatedAt, &a.RequestID, &a.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, domain.ErrNotFound
	}
	return a, err
}
