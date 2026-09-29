package controller

import (
	"net/http"
	"tvoydom/domain"
	"tvoydom/service"
)

func (h Handler) workflowRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PATCH /api/organization/requests/{id}/assignment", h.withID(h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		var in service.AssignmentInput
		if err := decodeJSON(r, &in); err != nil {
			badBody(w, err)
			return
		}
		item, err := (service.WorkflowService{Repo: h.Repo, Access: service.AccessService{Repo: h.Repo}}).UpdateAssignment(r.Context(), r.PathValue("id"), in)
		respond(w, item, err)
	})))
	mux.HandleFunc("POST /api/requests/{id}/resolution", h.withID(h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		var in service.ResidentDecisionInput
		if err := decodeJSON(r, &in); err != nil {
			badBody(w, err)
			return
		}
		item, err := (service.WorkflowService{Repo: h.Repo, Access: service.AccessService{Repo: h.Repo}}).DecideResolution(r.Context(), r.PathValue("id"), in)
		if err == nil {
			h.Service.Notifications.Notify(r.Context(), item)
			h.Service.NotifyManagers(r.Context(), item)
		}
		respond(w, item, err)
	})))
	mux.HandleFunc("PATCH /api/admin/requests/{id}/route", h.withID(h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		var in struct {
			PrimaryOrganizationID    string `json:"primaryOrganizationId"`
			ContractorOrganizationID string `json:"contractorOrganizationId"`
			Reason                   string `json:"reason"`
		}
		if err := decodeJSON(r, &in); err != nil {
			badBody(w, err)
			return
		}
		item, err := (service.WorkflowService{Repo: h.Repo, Access: service.AccessService{Repo: h.Repo}}).RouteManually(r.Context(), r.PathValue("id"), in.PrimaryOrganizationID, in.ContractorOrganizationID, in.Reason)
		respond(w, item, err)
	})))
}
