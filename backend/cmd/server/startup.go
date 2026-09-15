package main

import (
	"context"
	"time"
)

// Compose restart can start the API before Postgres has finished recovering.
func waitForDatabase(ctx context.Context, interval time.Duration, ping func(context.Context) error) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := ping(ctx); err == nil {
			return nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
