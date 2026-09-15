package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaitForDatabaseRetries(t *testing.T) {
	calls := 0
	err := waitForDatabase(context.Background(), time.Millisecond, func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("starting")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
func TestWaitForDatabaseStopsOnTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Millisecond)
	defer cancel()
	err := waitForDatabase(ctx, time.Millisecond, func(context.Context) error { return errors.New("unavailable") })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}
