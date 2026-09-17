package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tvoydom/gar-init/internal/config"
	"tvoydom/gar-init/internal/db"
	"tvoydom/gar-init/internal/importer"
)

func main() {
	logger := log.New(os.Stdout, "", 0)
	if err := run(logger); err != nil {
		logger.Printf("[GAR] error: %v", err)
		os.Exit(1)
	}
}

func run(logger *log.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		return fmt.Errorf("connect PostgreSQL: %w", err)
	}
	defer pool.Close()
	readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := pool.Ping(readyCtx); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	store := db.New(pool)
	if err := store.EnsureSchema(ctx); err != nil {
		return err
	}
	lock, err := store.AcquireLock(ctx)
	if err != nil {
		return err
	}
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer releaseCancel()
		if err := lock.Release(releaseCtx); err != nil {
			logger.Printf("[GAR] warning: %v", err)
		}
	}()
	return importer.Run(ctx, cfg, store, logger, time.Now)
}
