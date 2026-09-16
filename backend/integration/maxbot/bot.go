package maxbot

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type API interface {
	Updates(context.Context, *int64) (Batch, error)
	Send(context.Context, int64, int64, Message) error
}
type Bot struct {
	API              API
	AppURL, Username string
}

func ValidateSettings(appURL, username string) error {
	u, err := url.Parse(appURL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || len(appURL) > 2048 {
		return errors.New("MAX_APP_URL: укажите публичный HTTPS-адрес приложения")
	}
	host := strings.ToLower(u.Hostname())
	ip := net.ParseIP(host)
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || (ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified())) {
		return errors.New("MAX_APP_URL: локальный адрес не откроется на телефоне, нужен публичный HTTPS-адрес")
	}
	if username != "" && !regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(username) {
		return errors.New("MAX_BOT_USERNAME: укажите username без @ и ссылки")
	}
	return nil
}
func (b Bot) greeting() Message {
	buttons := [][]Button{}
	if b.Username != "" {
		buttons = append(buttons, []Button{{Type: "open_app", Text: "Открыть «Твой дом»", WebApp: b.Username}})
	}
	buttons = append(buttons, []Button{{Type: "link", Text: "Перейти на сайт", URL: b.AppURL}})
	return Message{Text: "Добро пожаловать в «Твой дом»!\n\nЗдесь можно оставить заявку в управляющую компанию, задать вопрос и следить за решением проблем дома.\n\nОткройте приложение по кнопке ниже.", Attachments: []Attachment{{Type: "inline_keyboard", Payload: Keyboard{Buttons: buttons}}}}
}
func (b Bot) Handle(ctx context.Context, u Update) error {
	chatID, userID := u.ChatID, u.User.ID
	switch u.Type {
	case "bot_started":
		if u.User.IsBot {
			return nil
		}
	case "message_created":
		if u.Message.Sender.IsBot || (u.Message.Recipient.ChatType != "" && u.Message.Recipient.ChatType != "dialog") {
			return nil
		}
		words := strings.Fields(u.Message.Body.Text)
		if len(words) == 0 {
			return nil
		}
		command := words[0]
		if command != "/start" && (b.Username == "" || !strings.EqualFold(command, "/start@"+b.Username)) {
			return nil
		}
		chatID, userID = u.Message.Recipient.ChatID, u.Message.Sender.ID
	default:
		return nil
	}
	if chatID == 0 && userID == 0 {
		return nil
	}
	return b.API.Send(ctx, chatID, userID, b.greeting())
}
func pause(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
func permanent(err error) bool {
	var apiErr APIError
	return errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 && apiErr.Status != 429
}

// Run is a single long-poll consumer for the MVP. Marker is kept in memory;
// restarting can replay the last event. Run only one instance per bot token.
func (b Bot) Run(ctx context.Context) error {
	var marker *int64
	for ctx.Err() == nil {
		batch, err := b.API.Updates(ctx, marker)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if permanent(err) {
				return err
			}
			slog.Warn("MAX polling retry", "error", err)
			if !pause(ctx, 5*time.Second) {
				return nil
			}
			continue
		}
		for _, update := range batch.Updates {
			backoff := time.Second
			for {
				err = b.Handle(ctx, update)
				if err == nil {
					break
				}
				if ctx.Err() != nil {
					return nil
				}
				var apiErr APIError
				if errors.As(err, &apiErr) && apiErr.Status == 401 {
					return err
				}
				if permanent(err) {
					slog.Warn("MAX reply rejected; skipping event", "error", err)
					break
				}
				slog.Warn("MAX reply retry", "error", err)
				if !pause(ctx, backoff) {
					return nil
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
			}
			// Stay below MAX's per-dialog send limit, also for duplicate start events.
			if !pause(ctx, 600*time.Millisecond) {
				return nil
			}
		}
		if batch.Marker != nil {
			marker = batch.Marker
		}
		if len(batch.Updates) == 0 && !pause(ctx, 500*time.Millisecond) {
			return nil
		}
	}
	return nil
}
