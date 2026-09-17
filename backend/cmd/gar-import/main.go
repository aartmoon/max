package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"
	"tvoydom/config"
	"tvoydom/integration/gar"
	"tvoydom/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	path := flag.String("path", "", "path to official GAR/FIAS XML zip")
	batchSize := flag.Int("batch-size", 1000, "records per database batch")
	flag.Parse()
	if *path == "" {
		slog.Error("missing required --path")
		os.Exit(2)
	}
	c := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, c.DatabaseURL)
	if err != nil {
		slog.Error("database configuration", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	repo := repository.Postgres{Pool: pool}
	if err := repo.Migrate(ctx); err != nil {
		slog.Error("database initialization", "error", err)
		os.Exit(1)
	}
	file, err := os.Open(*path)
	if err != nil {
		slog.Error("open GAR archive", "path", *path, "error", err)
		os.Exit(1)
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		slog.Error("stat GAR archive", "path", *path, "error", err)
		os.Exit(1)
	}
	started := time.Now()
	if err := gar.ImportZip(context.Background(), file, stat.Size(), repo, gar.Options{BatchSize: *batchSize}); err != nil {
		slog.Error("GAR import failed", "path", *path, "error", err)
		os.Exit(1)
	}
	slog.Info("GAR import finished", "path", *path, "duration", time.Since(started).String())
}
