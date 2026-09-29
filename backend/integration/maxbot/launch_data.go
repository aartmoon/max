package maxbot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Identity struct {
	UserID int64
	ChatID int64
	Name   string
}

func ValidateLaunchData(raw, token string, now time.Time) (Identity, error) {
	invalid := errors.New("некорректные данные запуска MAX")
	if raw == "" || token == "" || len(raw) > 16384 {
		return Identity{}, invalid
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return Identity{}, invalid
	}
	for _, value := range values {
		if len(value) != 1 {
			return Identity{}, invalid
		}
	}
	hashValues, ok := values["hash"]
	if !ok || len(hashValues) != 1 {
		return Identity{}, invalid
	}
	provided, err := hex.DecodeString(hashValues[0])
	if err != nil || len(provided) != sha256.Size {
		return Identity{}, invalid
	}
	keys := make([]string, 0, len(values)-1)
	for key := range values {
		if key != "hash" {
			keys = append(keys, key)
		}
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
	if !hmac.Equal(provided, signature.Sum(nil)) {
		return Identity{}, invalid
	}
	authUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return Identity{}, invalid
	}
	authDate := time.Unix(authUnix, 0)
	if authDate.Before(now.Add(-10*time.Minute)) || authDate.After(now.Add(time.Minute)) {
		return Identity{}, invalid
	}
	var user struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
	}
	var chat struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(values.Get("user")), &user) != nil || json.Unmarshal([]byte(values.Get("chat")), &chat) != nil || user.ID <= 0 || chat.ID <= 0 || chat.Type != "DIALOG" {
		return Identity{}, invalid
	}
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name == "" {
		name = strings.TrimSpace(user.Username)
	}
	if name == "" {
		name = "Пользователь MAX"
	}
	return Identity{UserID: user.ID, ChatID: chat.ID, Name: name}, nil
}
