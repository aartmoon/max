package maxbot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStartRepliesWithWebsiteAndMiniApp(t *testing.T) {
	for _, event := range []string{
		`{"update_type":"bot_started","chat_id":123,"user":{"user_id":45}}`,
		`{"update_type":"message_created","message":{"sender":{"user_id":45},"recipient":{"chat_id":123,"chat_type":"dialog"},"body":{"text":"/start demo"}}}`,
	} {
		t.Run(event, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "POST" || r.URL.Path != "/messages" || r.URL.Query().Get("chat_id") != "123" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "test-token" || r.URL.Query().Has("access_token") {
					t.Error("token must only be in header")
				}
				var body Message
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				buttons := body.Attachments[0].Payload.Buttons
				if body.Text == "" || len(buttons) != 2 || buttons[0][0].Type != "open_app" || buttons[0][0].WebApp != "tvoydom_bot" || buttons[1][0].URL != "https://home.example.org" {
					t.Errorf("unexpected greeting %+v", body)
				}
				w.Write([]byte(`{"message":{"body":{"mid":"1"}}}`))
			}))
			defer server.Close()
			bot := Bot{API: &Client{BaseURL: server.URL, Token: "test-token", HTTP: server.Client()}, AppURL: "https://home.example.org", Username: "tvoydom_bot"}
			var update Update
			json.Unmarshal([]byte(event), &update)
			if err := bot.Handle(context.Background(), update); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Errorf("calls=%d", calls)
			}
		})
	}
}
func TestNonStartAndBotMessagesIgnored(t *testing.T) {
	bot := Bot{} // No API call is permitted for unrelated messages.
	for _, event := range []string{
		`{"update_type":"message_created","message":{"body":{"text":"Привет"}}}`,
		`{"update_type":"message_created","message":{"body":{"text":"/starter"}}}`,
		`{"update_type":"message_created","message":{"sender":{"is_bot":true},"body":{"text":"/start"}}}`,
		`{"update_type":"message_callback"}`,
	} {
		var u Update
		json.Unmarshal([]byte(event), &u)
		if err := bot.Handle(context.Background(), u); err != nil {
			t.Fatal(err)
		}
	}
}
func TestPollingMarkerAndErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("marker") != "9007199254740993" || r.URL.Query().Get("types") != "bot_started,message_created" {
			t.Error(r.URL)
		}
		w.Write([]byte(`{"updates":[],"marker":9007199254740994}`))
	}))
	defer server.Close()
	client := Client{BaseURL: server.URL, Token: "secret", HTTP: server.Client()}
	marker := int64(9007199254740993)
	batch, err := client.Updates(context.Background(), &marker)
	if err != nil || batch.Marker == nil || *batch.Marker != 9007199254740994 {
		t.Fatalf("%+v %v", batch, err)
	}
}

func TestWebsiteOnlyAndSettings(t *testing.T) {
	msg := (Bot{AppURL: "https://home.example.org"}).greeting()
	if len(msg.Attachments[0].Payload.Buttons) != 1 || msg.Attachments[0].Payload.Buttons[0][0].Type != "link" {
		t.Fatal(msg)
	}
	for _, u := range []string{"", "http://example.org", "https://localhost:3000", "https://127.0.0.1", "https://10.0.0.1", "https://user:pass@example.org"} {
		if ValidateSettings(u, "") == nil {
			t.Errorf("accepted %s", u)
		}
	}
	if ValidateSettings("https://home.example.org", "tvoydom_bot") != nil {
		t.Fatal("valid settings rejected")
	}
	if ValidateSettings("https://home.example.org", "@bad") == nil {
		t.Fatal("invalid username accepted")
	}
}
func TestAPIErrorDoesNotLeakResponseOrToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte("private-token secret-response"))
	}))
	defer server.Close()
	c := Client{BaseURL: server.URL, Token: "private-token", HTTP: server.Client()}
	_, err := c.Updates(context.Background(), nil)
	if err == nil || strings.Contains(err.Error(), "private-token") || strings.Contains(err.Error(), "secret-response") {
		t.Fatalf("unsafe error %v", err)
	}
	if !permanent(err) {
		t.Fatal("401 must stop polling")
	}
	if permanent(APIError{Status: 429}) || permanent(APIError{Status: 503}) {
		t.Fatal("temporary failures should retry")
	}
}

type pollingAPI struct {
	calls  int
	t      *testing.T
	cancel context.CancelFunc
}

func (f *pollingAPI) Updates(ctx context.Context, marker *int64) (Batch, error) {
	f.calls++
	if f.calls == 1 {
		if marker != nil {
			f.t.Error("first marker should be absent")
		}
		m := int64(123)
		return Batch{Marker: &m}, nil
	}
	if marker == nil || *marker != 123 {
		f.t.Error("marker was not advanced")
	}
	f.cancel()
	return Batch{}, context.Canceled
}
func (f *pollingAPI) Send(context.Context, int64, int64, Message) error {
	f.t.Error("unexpected send")
	return nil
}
func TestRunnerAdvancesMarkerAndCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api := &pollingAPI{t: t, cancel: cancel}
	if err := (Bot{API: api}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if api.calls != 2 {
		t.Fatal(api.calls)
	}
}
