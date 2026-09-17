package db

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tvoydom/gar-init/internal/importer"
	"tvoydom/gar-init/internal/model"
)

//go:embed schema.sql
var schemaSQL string

//go:embed indexes.sql
var indexesSQL string

//go:embed search.sql
var searchSQL string

var ErrFileChanged = errors.New("completed GAR file changed")

const advisoryLockKey int64 = 0x474152494E4954

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) EnsureSchema(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("create GAR schema: %w", err)
	}
	return nil
}

type Lock struct {
	conn *pgxpool.Conn
}

func (s *Store) AcquireLock(ctx context.Context) (*Lock, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire GAR lock connection: %w", err)
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		conn.Release()
		return nil, fmt.Errorf("acquire GAR advisory lock: %w", err)
	}
	return &Lock{conn: conn}, nil
}

func (l *Lock) Release(ctx context.Context) error {
	if l == nil || l.conn == nil {
		return nil
	}
	_, err := l.conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, advisoryLockKey)
	l.conn.Release()
	l.conn = nil
	if err != nil {
		return fmt.Errorf("release GAR advisory lock: %w", err)
	}
	return nil
}

func (s *Store) Initialized(ctx context.Context) (bool, string, error) {
	var initialized bool
	var sourceType *string
	err := s.pool.QueryRow(ctx, `SELECT initialized, source_type FROM gar_import_metadata WHERE singleton`).Scan(&initialized, &sourceType)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("read GAR initialization state: %w", err)
	}
	if sourceType == nil {
		return initialized, "", nil
	}
	return initialized, *sourceType, nil
}

func (s *Store) ImportFile(ctx context.Context, source importer.SourceFile, parse importer.ParseFile) (count int64, skipped bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("begin %s import: %w", source.Name, err)
	}
	defer tx.Rollback(ctx)
	var storedSize, storedCount int64
	err = tx.QueryRow(ctx, `SELECT file_size, row_count FROM gar_import_state WHERE file_name=$1`, source.Name).Scan(&storedSize, &storedCount)
	switch {
	case err == nil && storedSize != source.Size:
		return 0, false, fmt.Errorf("%w: %s was %d bytes and is now %d", ErrFileChanged, source.Name, storedSize, source.Size)
	case err == nil:
		return storedCount, true, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return 0, false, fmt.Errorf("read state for %s: %w", source.Name, err)
	}
	started := time.Now().UTC()
	count, err = parse(ctx, func(ctx context.Context, family model.Family, rows [][]any) error {
		copied, err := tx.CopyFrom(ctx, pgx.Identifier{family.Table}, family.Columns(), pgx.CopyFromRows(rows))
		if err != nil {
			return err
		}
		if copied != int64(len(rows)) {
			return fmt.Errorf("COPY %s wrote %d of %d rows", family.Table, copied, len(rows))
		}
		return nil
	})
	if err != nil {
		return count, false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO gar_import_state(file_name,entity_type,file_size,row_count,started_at,completed_at) VALUES($1,$2,$3,$4,$5,now())`, source.Name, source.Family.Key, source.Size, count, started); err != nil {
		return count, false, fmt.Errorf("record state for %s: %w", source.Name, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return count, false, fmt.Errorf("commit %s import: %w", source.Name, err)
	}
	return count, false, nil
}

func (s *Store) Finalize(ctx context.Context, sourceType string, sourceDate time.Time) error {
	if sourceType != "xml" && sourceType != "demo" {
		return fmt.Errorf("invalid GAR source type %q", sourceType)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin GAR finalization: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, indexesSQL); err != nil {
		return fmt.Errorf("create GAR indexes: %w", err)
	}
	if _, err := tx.Exec(ctx, searchSQL); err != nil {
		return fmt.Errorf("build GAR search addresses: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO gar_import_metadata(singleton,initialized,initialized_at,source_date,source_type) VALUES(true,true,now(),$1,$2) ON CONFLICT(singleton) DO UPDATE SET initialized=true,initialized_at=EXCLUDED.initialized_at,source_date=EXCLUDED.source_date,source_type=EXCLUDED.source_type`, sourceDate, sourceType); err != nil {
		return fmt.Errorf("record GAR completion: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit GAR finalization: %w", err)
	}
	return nil
}
