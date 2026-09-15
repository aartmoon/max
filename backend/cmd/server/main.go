package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"tvoydom/config"
	"tvoydom/controller"
	"tvoydom/integration"
	"tvoydom/repository"
	"tvoydom/service"
)

func main() {
	c := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, c.DatabaseURL)
	if err != nil {
		slog.Error("database configuration", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err = waitForDatabase(ctx, time.Second, pool.Ping); err != nil {
		slog.Error("database readiness timeout", "error", err)
		os.Exit(1)
	}
	repo := repository.Postgres{Pool: pool}
	if err = repo.Migrate(ctx); err != nil {
		slog.Error("database initialization", "error", err)
		os.Exit(1)
	}
	svc := service.RequestService{Repo: repo, Classifier: service.RuleClassifier{}, Router: service.RuleRouter{}, Notifications: service.NotificationService{Client: integration.MockMaxClient{}}, Housing: integration.MockHousingSystemGateway{}}
	server := &http.Server{Addr: ":" + c.Port, Handler: (controller.Handler{Service: svc, Repo: repo, MockStatusEnabled: c.MockStatusEnabled}).Routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	stop, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()
	go func() {
		<-stop.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()
	slog.Info("server started", "port", c.Port)
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("http server", "error", err)
		os.Exit(1)
	}
}
