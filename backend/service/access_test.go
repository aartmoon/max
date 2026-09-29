package service

import (
	"context"
	"errors"
	"testing"

	"tvoydom/domain"
)

type accessRepoStub struct{ access domain.RequestAccess }

func (r accessRepoStub) RequestAccess(context.Context, string) (domain.RequestAccess, error) {
	return r.access, nil
}

func TestRequestAccessSeparatesOrganizations(t *testing.T) {
	access := domain.RequestAccess{RequestID: "7", OwnerUserID: "1", PrimaryOrganizationID: "10", ContractorOrganizationID: "20", AssigneeUserID: "30", AssigneeOrganizationID: "30", Status: "IN_PROGRESS"}
	for _, tc := range []struct {
		name       string
		user       domain.User
		capability Capability
		allowed    bool
	}{
		{"owner", domain.User{ID: "1", Roles: []string{"resident"}}, CapabilityRead, true},
		{"other resident", domain.User{ID: "2", Roles: []string{"resident"}}, CapabilityRead, false},
		{"primary manager", domain.User{ID: "3", Roles: []string{"manager"}, OrganizationID: stringPointer("10")}, CapabilityManage, true},
		{"contractor manager", domain.User{ID: "4", Roles: []string{"manager"}, OrganizationID: stringPointer("20")}, CapabilityInternal, true},
		{"foreign manager", domain.User{ID: "5", Roles: []string{"manager"}, OrganizationID: stringPointer("99")}, CapabilityRead, false},
		{"admin", domain.User{ID: "6", Roles: []string{"admin"}}, CapabilityManage, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := (AccessService{Repo: accessRepoStub{access}}).Require(WithCurrentUser(context.Background(), tc.user), "7", tc.capability)
			if tc.allowed && err != nil {
				t.Fatal(err)
			}
			if !tc.allowed && !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func stringPointer(value string) *string { return &value }
