package service

import (
	"context"
	"testing"

	"tvoydom/domain"
	"tvoydom/integration/maxbot"
)

type maxNotificationRepoStub struct {
	owner    MAXDestination
	managers []MAXDestination
}

func (r maxNotificationRepoStub) RequestOwnerMAXDestination(context.Context, string) (MAXDestination, bool, error) {
	return r.owner, r.owner.UserID != 0, nil
}

func (r maxNotificationRepoStub) ManagerMAXDestinations(context.Context, string) ([]MAXDestination, error) {
	return r.managers, nil
}

type maxSendCapture struct {
	calls []struct {
		chatID, userID int64
		message        maxbot.Message
	}
}

func (c *maxSendCapture) Send(_ context.Context, chatID, userID int64, message maxbot.Message) error {
	c.calls = append(c.calls, struct {
		chatID, userID int64
		message        maxbot.Message
	}{chatID, userID, message})
	return nil
}

func TestMAXNotificationSendsOwnerHumanStatusAndRequestLink(t *testing.T) {
	sender := &maxSendCapture{}
	svc := MAXNotificationService{
		Repo: maxNotificationRepoStub{owner: MAXDestination{UserID: 7, ChatID: 8}},
		API:  sender, Username: "tvoydom_bot", AppURL: "https://home.example.org",
	}
	svc.NotifyOwner(context.Background(), domain.Request{ID: "42", Status: "IN_PROGRESS"}, MAXEvent{Kind: MAXEventStatus})

	if len(sender.calls) != 1 || sender.calls[0].chatID != 8 || sender.calls[0].userID != 7 {
		t.Fatalf("unexpected sends: %+v", sender.calls)
	}
	message := sender.calls[0].message
	if message.Text != "Обращение №42\nСтатус: В работе" {
		t.Fatalf("unexpected text: %q", message.Text)
	}
	button := message.Attachments[0].Payload.Buttons[0][0]
	if button.Type != "link" || button.URL != "https://max.ru/tvoydom_bot?startapp=request_42" {
		t.Fatalf("unexpected button: %+v", button)
	}
}

func TestMAXNotificationDeduplicatesManagersAndSkipsUnlinkedOwner(t *testing.T) {
	sender := &maxSendCapture{}
	destination := MAXDestination{UserID: 10, ChatID: 11}
	svc := MAXNotificationService{Repo: maxNotificationRepoStub{managers: []MAXDestination{destination, destination}}, API: sender, AppURL: "https://home.example.org"}

	svc.NotifyOwner(context.Background(), domain.Request{ID: "5"}, MAXEvent{Kind: MAXEventMessage, Text: "ответ"})
	svc.NotifyManagers(context.Background(), domain.Request{ID: "5"}, MAXEvent{Kind: MAXEventMessage, Text: "  Очень длинное сообщение жителя  "})

	if len(sender.calls) != 1 {
		t.Fatalf("duplicate or missing sends: %+v", sender.calls)
	}
	if sender.calls[0].message.Text != "Обращение №5\nНовое сообщение: Очень длинное сообщение жителя" {
		t.Fatalf("unexpected text: %q", sender.calls[0].message.Text)
	}
	button := sender.calls[0].message.Attachments[0].Payload.Buttons[0][0]
	if button.URL != "https://home.example.org" {
		t.Fatalf("unexpected fallback link: %+v", button)
	}
}
