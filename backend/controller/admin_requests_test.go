package controller

import (
	"net/url"
	"testing"
	"time"

	"tvoydom/domain"
)

func TestParseAdminRequestQueryDefaultsAndManagerScope(t *testing.T) {
	org := "7"
	user := domain.User{ID: "9", Roles: []string{"manager"}, OrganizationID: &org}
	got, err := parseAdminRequestQuery(url.Values{"organization": {"999"}}, user, time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if got.Page != 1 || got.PageSize != 20 || got.Queue != "ACTIVE" || got.Sort != "PRIORITY" || got.OrganizationID != "7" || got.UserID != "9" {
		t.Fatalf("unexpected defaults/scope: %+v", got)
	}
}

func TestParseAdminRequestQueryRejectsInvalidValues(t *testing.T) {
	for _, values := range []url.Values{
		{"page": {"0"}}, {"pageSize": {"25"}}, {"queue": {"BAD"}},
		{"status": {"BAD"}}, {"kind": {"BAD"}}, {"sort": {"BAD"}},
	} {
		if _, err := parseAdminRequestQuery(values, domain.User{ID: "1", Roles: []string{"admin"}}, time.Now()); err == nil {
			t.Fatalf("invalid values accepted: %v", values)
		}
	}
}
