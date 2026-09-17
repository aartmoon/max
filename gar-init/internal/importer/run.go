package importer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"tvoydom/gar-init/internal/config"
)

type Store interface {
	Initialized(context.Context) (bool, string, error)
	ImportFile(context.Context, SourceFile, ParseFile) (count int64, skipped bool, err error)
	SeedDemo(context.Context) error
	Finalize(context.Context, string, time.Time) error
}

var snapshotDatePattern = regexp.MustCompile(`_(\d{8})_`)

func Run(ctx context.Context, cfg config.Config, store Store, logger *log.Logger, now func() time.Time) error {
	initialized, _, err := store.Initialized(ctx)
	if err != nil {
		return err
	}
	if initialized {
		logger.Printf("[GAR] GAR database already initialized")
		return nil
	}
	files, err := Discover(cfg.DataDir)
	if err != nil {
		if errors.Is(err, ErrNoSupportedFiles) && cfg.AllowDemoData {
			logger.Printf("[GAR] no XML files found; using demo address data")
			if err := store.SeedDemo(ctx); err != nil {
				return fmt.Errorf("seed GAR demo data: %w", err)
			}
			if err := store.Finalize(ctx, "demo", now().UTC()); err != nil {
				return err
			}
			logger.Printf("[GAR] demo address data initialized")
			return nil
		}
		return err
	}
	for _, source := range files {
		label := strings.TrimSuffix(source.Family.Prefix, "_")
		started := now()
		logger.Printf("[GAR] importing %s", source.Name)
		count, skipped, err := store.ImportFile(ctx, source, func(ctx context.Context, write BatchWriter) (int64, error) {
			file, err := os.Open(source.Path)
			if err != nil {
				return 0, fmt.Errorf("open %s: %w", source.Path, err)
			}
			defer file.Close()
			return Parse(ctx, file, source.Family, cfg.BatchSize, cfg.LogEvery, write, func(rows int64) {
				logger.Printf("[GAR] %s: %d rows", label, rows)
			})
		})
		if err != nil {
			return fmt.Errorf("import %s: %w", source.Name, err)
		}
		if skipped {
			logger.Printf("[GAR] skipping %s: already imported (%d rows)", source.Name, count)
			continue
		}
		logger.Printf("[GAR] finished %s: %d rows in %.1fs", label, count, now().Sub(started).Seconds())
	}
	if err := store.Finalize(ctx, "xml", snapshotDate(files, now().UTC())); err != nil {
		return err
	}
	logger.Printf("[GAR] initialization completed")
	return nil
}

func snapshotDate(files []SourceFile, fallback time.Time) time.Time {
	for _, file := range files {
		match := snapshotDatePattern.FindStringSubmatch(file.Name)
		if len(match) != 2 {
			continue
		}
		if date, err := time.Parse("20060102", match[1]); err == nil {
			return date
		}
	}
	return fallback
}
