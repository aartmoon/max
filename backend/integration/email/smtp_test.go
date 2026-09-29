package email

import (
	"mime"
	"strings"
	"testing"
)

func TestBuildLoginCodeMessageUsesStyledHTMLWithTextFallback(t *testing.T) {
	message := string(buildLoginCodeMessage("noreply@example.com", "user@example.com", "123456"))

	for _, want := range []string{
		"Subject: =?UTF-8?",
		"Content-Type: multipart/alternative;",
		"text/plain; charset=utf-8",
		"text/html; charset=utf-8",
		"#471aff",
		"#00bfff",
		"123456",
		"MAX",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("message does not contain %q:\n%s", want, message)
		}
	}
	if !strings.Contains(message, "Код действует 10 минут.") {
		t.Fatal("plain text fallback is missing the ttl copy")
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(headerValue(message, "Subject"))
	if err != nil {
		t.Fatal(err)
	}
	if subject != "Код для входа" {
		t.Fatalf("unexpected subject: %q", subject)
	}
}

func TestBuildTextMessageEncodesRussianSubject(t *testing.T) {
	message := string(buildTextMessage("noreply@example.com", "user@example.com", "Новая заявка", "body"))

	subjectHeader := headerValue(message, "Subject")
	if !strings.Contains(subjectHeader, "=?UTF-8?") {
		t.Fatalf("subject is not encoded: %q", subjectHeader)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(subjectHeader)
	if err != nil {
		t.Fatal(err)
	}
	if subject != "Новая заявка" {
		t.Fatalf("unexpected subject: %q", subject)
	}
}

func headerValue(message, name string) string {
	prefix := name + ": "
	for _, line := range strings.Split(message, "\r\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}
