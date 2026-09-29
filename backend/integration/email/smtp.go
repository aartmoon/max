package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"html"
	"log/slog"
	"mime"
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
	return s.sendMessage(ctx, email, buildLoginCodeMessage(s.from(), email, code))
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

func (s SMTPSender) from() string {
	if s.Config.From != "" {
		return s.Config.From
	}
	return s.Config.Username
}

func (s SMTPSender) send(ctx context.Context, to, subject, body string) error {
	return s.sendMessage(ctx, to, buildTextMessage(s.from(), to, subject, body))
}

func (s SMTPSender) sendMessage(ctx context.Context, to string, message []byte) error {
	cfg := s.Config
	from := s.from()
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	errc := make(chan error, 1)
	go func() {
		if cfg.TLS {
			errc <- sendTLS(addr, auth, from, []string{to}, message, cfg.Host)
			return
		}
		errc <- smtp.SendMail(addr, auth, from, []string{to}, message)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errc:
		return err
	}
}

func buildTextMessage(from, to, subject, body string) []byte {
	return []byte(strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + encodeHeader(subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n"))
}

func buildLoginCodeMessage(from, to, code string) []byte {
	const boundary = "tvoy-dom-max-login"
	text := "Ваш код входа: " + code + "\n\nКод действует 10 минут."
	safeCode := html.EscapeString(code)
	htmlBody := `<!doctype html>
<html lang="ru">
<body style="margin:0;padding:0;background:#f7f5ff;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Arial,sans-serif;color:#0d001a;">
  <div style="padding:28px 16px;">
    <div style="max-width:480px;margin:0 auto;background:#ffffff;border:1px solid #ddd4ff;border-radius:22px;overflow:hidden;">
      <div style="padding:22px 24px;background:linear-gradient(135deg,#471aff,#9500ff 62%,#00bfff);color:#ffffff;">
        <div style="font-size:12px;font-weight:700;letter-spacing:1.4px;text-transform:uppercase;">MAX · Твой дом</div>
        <h1 style="margin:12px 0 0;font-size:24px;line-height:1.25;font-weight:750;">Код для входа</h1>
      </div>
      <div style="padding:26px 24px 28px;">
        <p style="margin:0 0 18px;font-size:15px;line-height:1.6;color:#675f7a;">Введите этот код в приложении, чтобы войти в кабинет жителя.</p>
        <div style="font-size:34px;line-height:1;letter-spacing:8px;font-weight:800;color:#471aff;background:#f2edff;border-radius:16px;padding:18px 20px;text-align:center;">` + safeCode + `</div>
        <p style="margin:18px 0 0;font-size:13px;line-height:1.6;color:#675f7a;">Код действует 10 минут. Если вы не запрашивали вход, просто удалите это письмо.</p>
      </div>
    </div>
  </div>
</body>
</html>`
	return []byte(strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + encodeHeader("Код для входа"),
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=" + boundary,
		"",
		"--" + boundary,
		"Content-Type: text/plain; charset=utf-8",
		"",
		text,
		"--" + boundary,
		"Content-Type: text/html; charset=utf-8",
		"",
		htmlBody,
		"--" + boundary + "--",
		"",
	}, "\r\n"))
}

func encodeHeader(value string) string {
	return mime.QEncoding.Encode("UTF-8", value)
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
