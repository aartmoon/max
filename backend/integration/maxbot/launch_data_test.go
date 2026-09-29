package maxbot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

func signedLaunchData(token string, values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+values.Get(key))
	}
	keyMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = keyMAC.Write([]byte(token))
	signature := hmac.New(sha256.New, keyMAC.Sum(nil))
	_, _ = signature.Write([]byte(strings.Join(lines, "\n")))
	values.Set("hash", hex.EncodeToString(signature.Sum(nil)))
	return values.Encode()
}

func TestValidateLaunchDataAcceptsAuthenticDialog(t *testing.T) {
	now := time.Unix(1771409719, 0)
	raw := signedLaunchData("bot-token", url.Values{
		"auth_date": {"1771409719"},
		"chat":      {`{"id":12345,"type":"DIALOG"}`},
		"query_id":  {"query-1"},
		"user":      {`{"id":67890,"first_name":"Иван"}`},
	})

	identity, err := ValidateLaunchData(raw, "bot-token", now)
	if err != nil {
		t.Fatal(err)
	}
	if identity.UserID != 67890 || identity.ChatID != 12345 {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestValidateLaunchDataRejectsUntrustedInput(t *testing.T) {
	now := time.Unix(1771409719, 0)
	valid := func() string {
		return signedLaunchData("bot-token", url.Values{
			"auth_date": {"1771409719"},
			"chat":      {`{"id":12345,"type":"DIALOG"}`},
			"user":      {`{"id":67890}`},
		})
	}
	for name, raw := range map[string]string{
		"tampered":  strings.Replace(valid(), "67890", "67891", 1),
		"duplicate": valid() + "&user=%7B%22id%22%3A1%7D",
		"stale": signedLaunchData("bot-token", url.Values{
			"auth_date": {"1771409000"}, "chat": {`{"id":12345,"type":"DIALOG"}`}, "user": {`{"id":67890}`},
		}),
		"group": signedLaunchData("bot-token", url.Values{
			"auth_date": {"1771409719"}, "chat": {`{"id":12345,"type":"CHAT"}`}, "user": {`{"id":67890}`},
		}),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateLaunchData(raw, "bot-token", now); err == nil {
				t.Fatal("untrusted launch data accepted")
			}
		})
	}
}
