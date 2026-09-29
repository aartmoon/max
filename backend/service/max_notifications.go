package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"tvoydom/domain"
	"tvoydom/integration/maxbot"
)

type MAXDestination struct {
	UserID int64
	ChatID int64
}

type MAXNotificationRepository interface {
	RequestOwnerMAXDestination(context.Context, string) (MAXDestination, bool, error)
	ManagerMAXDestinations(context.Context, string) ([]MAXDestination, error)
}

type MAXMessageAPI interface {
	Send(context.Context, int64, int64, maxbot.Message) error
}

type MAXEventKind string

const (
	MAXEventCreated    MAXEventKind = "created"
	MAXEventStatus     MAXEventKind = "status"
	MAXEventMessage    MAXEventKind = "message"
	MAXEventAssignment MAXEventKind = "assignment"
	MAXEventRouting    MAXEventKind = "routing"
)

type MAXEvent struct {
	Kind MAXEventKind
	Text string
}

type MAXNotificationService struct {
	Repo             MAXNotificationRepository
	API              MAXMessageAPI
	Username, AppURL string
}

func (s MAXNotificationService) NotifyOwner(ctx context.Context, request domain.Request, event MAXEvent) {
	if s.Repo == nil || s.API == nil {
		return
	}
	destination, ok, err := s.Repo.RequestOwnerMAXDestination(ctx, request.ID)
	if err != nil {
		slog.Error("MAX owner lookup failed", "requestId", request.ID, "error", err)
		return
	}
	if ok {
		s.send(ctx, request, event, destination)
	}
}

func (s MAXNotificationService) NotifyManagers(ctx context.Context, request domain.Request, event MAXEvent) {
	if s.Repo == nil || s.API == nil {
		return
	}
	destinations, err := s.Repo.ManagerMAXDestinations(ctx, request.ID)
	if err != nil {
		slog.Error("MAX manager lookup failed", "requestId", request.ID, "error", err)
		return
	}
	seen := map[MAXDestination]bool{}
	for _, destination := range destinations {
		if destination.UserID == 0 || seen[destination] {
			continue
		}
		seen[destination] = true
		s.send(ctx, request, event, destination)
	}
}

func (s MAXNotificationService) send(ctx context.Context, request domain.Request, event MAXEvent, destination MAXDestination) {
	message := maxbot.Message{Text: formatMAXNotification(request, event)}
	link := s.AppURL
	if s.Username != "" {
		link = "https://max.ru/" + s.Username + "?startapp=request_" + request.ID
	}
	if link != "" {
		message.Attachments = []maxbot.Attachment{{Type: "inline_keyboard", Payload: maxbot.Keyboard{Buttons: [][]maxbot.Button{{{Type: "link", Text: "Открыть обращение", URL: link}}}}}}
	}
	if err := s.API.Send(ctx, destination.ChatID, destination.UserID, message); err != nil {
		slog.Error("MAX notification failed", "requestId", request.ID, "error", err)
	}
}

func formatMAXNotification(request domain.Request, event MAXEvent) string {
	prefix := "Обращение №" + request.ID + "\n"
	switch event.Kind {
	case MAXEventCreated:
		if strings.TrimSpace(event.Text) != "" {
			return prefix + maxExcerpt(event.Text, 180)
		}
		return prefix + "Заявка создана"
	case MAXEventStatus:
		return prefix + "Статус: " + maxStatusName(request.Status)
	case MAXEventMessage:
		text := maxExcerpt(event.Text, 180)
		if text == "" {
			text = "вложение"
		}
		return prefix + "Новое сообщение: " + text
	case MAXEventAssignment:
		return prefix + "Обновлено назначение: " + maxExcerpt(event.Text, 180)
	case MAXEventRouting:
		return prefix + "Назначена ответственная организация: " + maxExcerpt(event.Text, 180)
	default:
		return fmt.Sprintf("%sОбновление по заявке", prefix)
	}
}

func maxStatusName(status string) string {
	if name := map[string]string{
		"ROUTING_REQUIRED": "Требуется маршрутизация", "CREATED": "Создано", "SENT": "Отправлено",
		"ACCEPTED": "Принято", "IN_PROGRESS": "В работе", "RESOLVED": "Ожидает подтверждения",
		"CLOSED": "Закрыто", "REJECTED": "Отклонено",
	}[status]; name != "" {
		return name
	}
	return status
}

func maxExcerpt(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:limit])) + "…"
}
