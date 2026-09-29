package service

import (
	"context"

	"tvoydom/domain"
)

type Capability string

const (
	CapabilityRead     Capability = "read"
	CapabilityPublic   Capability = "write_public"
	CapabilityInternal Capability = "write_internal"
	CapabilityManage   Capability = "manage_work"
)

type AccessRepository interface {
	RequestAccess(context.Context, string) (domain.RequestAccess, error)
}

type AccessService struct{ Repo AccessRepository }

func (s AccessService) Require(ctx context.Context, requestID string, capability Capability) (domain.RequestAccess, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return domain.RequestAccess{}, domain.ErrUnauthorized
	}
	access, err := s.Repo.RequestAccess(ctx, requestID)
	if err != nil {
		return access, err
	}
	if HasRole(user, "admin") {
		return access, nil
	}
	if user.ID == access.OwnerUserID {
		if capability == CapabilityRead || capability == CapabilityPublic {
			return access, nil
		}
	}
	if HasRole(user, "manager") && user.OrganizationID != nil {
		organizationID := *user.OrganizationID
		if organizationID == access.PrimaryOrganizationID || organizationID == access.ContractorOrganizationID || organizationID == access.AssigneeOrganizationID {
			return access, nil
		}
	}
	return access, domain.ErrForbidden
}
