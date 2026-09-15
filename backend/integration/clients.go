package integration

import (
	"context"
	"log/slog"
	"tvoydom/domain"
)

type MaxClient interface {
	Notify(context.Context, string, string) error
}
type HousingSystemGateway interface {
	Submit(context.Context, domain.Request) error
}
type MockMaxClient struct{}

func (MockMaxClient) Notify(_ context.Context, userID, message string) error {
	slog.Info("mock MAX notification", "userId", userID, "event", message)
	return nil
}

type MockHousingSystemGateway struct{}

func (MockHousingSystemGateway) Submit(_ context.Context, r domain.Request) error {
	slog.Info("mock housing submission", "requestId", r.ID)
	return nil
}
