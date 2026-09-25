package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"tvoydom/domain"
)

type Sender interface {
	SendLoginCode(ctx context.Context, email, code string) error
	SendNewRequest(ctx context.Context, email string, r domain.Request) error
	SendRequestStatusChanged(ctx context.Context, email string, r domain.Request, comment string) error
}

type LogSender struct{}

func (LogSender) SendLoginCode(_ context.Context, email, code string) error {
	slog.Info("login code email", "email", email, "code", code)
	return nil
}

func (LogSender) SendNewRequest(_ context.Context, email string, r domain.Request) error {
	slog.Info("new request email", "email", email, "requestId", r.ID)
	return nil
}

func (LogSender) SendRequestStatusChanged(_ context.Context, email string, r domain.Request, comment string) error {
	slog.Info("request status email", "email", email, "requestId", r.ID, "status", r.Status, "comment", comment)
	return nil
}

type SMTPConfig struct {
	Host, Port string
	Username   string
	Password   string
	From       string
	TLS        bool
}

type SMTPSender struct {
	Config SMTPConfig
}

func (s SMTPSender) SendLoginCode(ctx context.Context, email, code string) error {
	return s.send(ctx, email, "Код входа в Твой дом", "Ваш код входа: "+code+"\n\nКод действует 10 минут.")
}

func (s SMTPSender) SendNewRequest(ctx context.Context, email string, r domain.Request) error {
	body := fmt.Sprintf("Новая заявка №%s\n\nАдрес: %s\nТип: %s\n\n%s", r.ID, r.Address, r.Kind, r.Description)
	return s.send(ctx, email, "Новая заявка №"+r.ID, body)
}

func (s SMTPSender) SendRequestStatusChanged(ctx context.Context, email string, r domain.Request, comment string) error {
	body := fmt.Sprintf("Статус заявки №%s изменён на: %s\n\nАдрес: %s", r.ID, r.Status, r.Address)
	if strings.TrimSpace(comment) != "" {
		body += "\n\nКомментарий: " + strings.TrimSpace(comment)
	}
	return s.send(ctx, email, "Статус заявки №"+r.ID+": "+r.Status, body)
}

func (s SMTPSender) send(ctx context.Context, to, subject, body string) error {
	cfg := s.Config
	if cfg.From == "" {
		cfg.From = cfg.Username
	}
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	message := strings.Join([]string{
		"From: " + cfg.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n")
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	errc := make(chan error, 1)
	go func() {
		if cfg.TLS {
			errc <- sendTLS(addr, auth, cfg.From, []string{to}, []byte(message), cfg.Host)
			return
		}
		errc <- smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(message))
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errc:
		return err
	}
}

func sendTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte, serverName string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12})
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, serverName)
	if err != nil {
		return err
	}
	defer client.Quit()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		w.Close()
		return err
	}
	return w.Close()
}
