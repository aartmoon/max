package db

import (
	"context"
	"errors"
	"fmt"
)

var ErrDatabaseNotEmpty = errors.New("GAR database is not empty")

func (s *Store) SeedDemo(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin GAR demo seed: %w", err)
	}
	defer tx.Rollback(ctx)
	var hasRows bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(
        SELECT 1 FROM gar_import_state
        UNION ALL SELECT 1 FROM gar_address_objects
        UNION ALL SELECT 1 FROM gar_houses
        UNION ALL SELECT 1 FROM gar_apartments
        UNION ALL SELECT 1 FROM gar_search_addresses
    )`).Scan(&hasRows); err != nil {
		return fmt.Errorf("check GAR database before demo seed: %w", err)
	}
	if hasRows {
		return ErrDatabaseNotEmpty
	}
	if err := loadDemoFixtures(ctx, tx); err != nil {
		return fmt.Errorf("seed GAR demo data: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit GAR demo seed: %w", err)
	}
	return nil
}
