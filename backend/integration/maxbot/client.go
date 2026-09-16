// Package maxbot contains the real MAX chat-bot adapter. It is independent of
// mock request notifications: demo user IDs must never be sent to MAX.
package maxbot

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const APIBaseURL = "https://platform-api2.max.ru"

type Client struct {
	BaseURL, Token string
	HTTP           *http.Client
}
type APIError struct{ Status int }

func (e APIError) Error() string { return fmt.Sprintf("MAX API: HTTP %d", e.Status) }
func NewClient(token, caFile string) (*Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, errors.New("не удалось прочитать MAX_CA_CERT_FILE")
		}
		roots, err := x509.SystemCertPool()
		if err != nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(pem) {
			return nil, errors.New("MAX_CA_CERT_FILE должен содержать сертификаты PEM")
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	}
	return &Client{BaseURL: APIBaseURL, Token: token, HTTP: &http.Client{Transport: transport, Timeout: 40 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) request(ctx context.Context, method, path string, query url.Values, body, out any) error {
	var payload io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path+"?"+query.Encode(), payload)
	if err != nil {
		return errors.New("некорректный адрес MAX API")
	}
	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Content-Type", "application/json")
	response, err := c.HTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("MAX недоступен: проверьте сеть и доверенные TLS-сертификаты")
	}
	defer response.Body.Close()
	// Never log response bodies: remote errors may contain sensitive data.
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return APIError{Status: response.StatusCode}
	}
	if out != nil {
		if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(out); err != nil {
			return errors.New("некорректный JSON в ответе MAX")
		}
	}
	return nil
}

type User struct {
	ID    int64 `json:"user_id"`
	IsBot bool  `json:"is_bot"`
}
type Update struct {
	Type    string `json:"update_type"`
	ChatID  int64  `json:"chat_id"`
	User    User   `json:"user"`
	Message struct {
		Sender    User `json:"sender"`
		Recipient struct {
			ChatID   int64  `json:"chat_id"`
			ChatType string `json:"chat_type"`
		} `json:"recipient"`
		Body struct {
			Text string `json:"text"`
		} `json:"body"`
	} `json:"message"`
}
type Batch struct {
	Updates []Update `json:"updates"`
	Marker  *int64   `json:"marker"`
}

func (c *Client) Updates(ctx context.Context, marker *int64) (Batch, error) {
	query := url.Values{"timeout": {"30"}, "limit": {"100"}, "types": {"bot_started,message_created"}}
	if marker != nil {
		query.Set("marker", strconv.FormatInt(*marker, 10))
	}
	var batch Batch
	err := c.request(ctx, http.MethodGet, "/updates", query, nil, &batch)
	return batch, err
}

type Button struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	URL    string `json:"url,omitempty"`
	WebApp string `json:"web_app,omitempty"`
}
type Keyboard struct {
	Buttons [][]Button `json:"buttons"`
}
type Attachment struct {
	Type    string   `json:"type"`
	Payload Keyboard `json:"payload"`
}
type Message struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

func (c *Client) Send(ctx context.Context, chatID, userID int64, message Message) error {
	query := url.Values{}
	if chatID != 0 {
		query.Set("chat_id", strconv.FormatInt(chatID, 10))
	} else if userID != 0 {
		query.Set("user_id", strconv.FormatInt(userID, 10))
	} else {
		return errors.New("MAX update без получателя")
	}
	return c.request(ctx, http.MethodPost, "/messages", query, message, nil)
}
