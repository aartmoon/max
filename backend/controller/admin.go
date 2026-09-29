package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
	"tvoydom/domain"
	"tvoydom/service"
)

// Explicitly a demo console, with no authentication or organization isolation yet.
func (h Handler) adminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/organizations", h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		v, e := h.Repo.Organizations(r.Context())
		if e == nil && !service.HasRole(user, "admin") {
			if user.OrganizationID == nil {
				respond(w, nil, domain.ErrForbidden)
				return
			}
			filtered := v[:0]
			for _, organization := range v {
				if organization.ID == *user.OrganizationID {
					filtered = append(filtered, organization)
				}
			}
			v = filtered
		}
		respond(w, v, e)
	}))
	mux.HandleFunc("POST /api/admin/organizations", h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		r.Body = http.MaxBytesReader(w, r.Body, 16384)
		var in service.SaveOrganizationInput
		if err := decodeJSON(r, &in); err != nil {
			badBody(w, err)
			return
		}
		v, e := (service.RoutingAdminService{Repo: h.Repo}).SaveOrganization(r.Context(), "", in)
		if e != nil {
			respond(w, nil, e)
			return
		}
		writeJSON(w, http.StatusCreated, v)
	}))
	mux.HandleFunc("GET /api/admin/users", h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		v, e := h.Repo.ListUsers(r.Context())
		respond(w, v, e)
	}))
	mux.HandleFunc("PATCH /api/admin/users/{id}/roles", h.withID(h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		var in struct {
			Roles []string `json:"roles"`
		}
		if err := decodeJSON(r, &in); err != nil {
			badBody(w, err)
			return
		}
		v, e := h.Repo.SetUserRoles(r.Context(), r.PathValue("id"), in.Roles)
		respond(w, v, e)
	})))
	mux.HandleFunc("PATCH /api/admin/users/{id}/organization", h.withID(h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		var in struct {
			OrganizationID string `json:"organizationId"`
		}
		if err := decodeJSON(r, &in); err != nil {
			badBody(w, err)
			return
		}
		v, e := h.Repo.SetUserOrganization(r.Context(), r.PathValue("id"), in.OrganizationID)
		respond(w, v, e)
	})))
	mux.HandleFunc("GET /api/admin/requests", h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		organizationID := ""
		if !service.HasRole(user, "admin") && user.OrganizationID == nil {
			respond(w, nil, domain.ErrForbidden)
			return
		}
		if user.OrganizationID != nil {
			organizationID = *user.OrganizationID
		}
		queue := r.URL.Query().Get("queue")
		if queue == "" {
			queue = "ACTIVE"
		}
		start, end := time.Now().UTC().Truncate(24*time.Hour), time.Now().UTC().Truncate(24*time.Hour).Add(24*time.Hour)
		if raw := r.URL.Query().Get("visitStart"); raw != "" {
			if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
				start = parsed
			}
		}
		if raw := r.URL.Query().Get("visitEnd"); raw != "" {
			if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
				end = parsed
			}
		}
		v, e := h.Repo.ListAccessibleRequests(r.Context(), organizationID, user.ID, queue, start, end)
		respond(w, v, e)
	}))
	mux.HandleFunc("GET /api/admin/requests/{id}", h.withID(h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		_, e := (service.AccessService{Repo: h.Repo}).Require(r.Context(), r.PathValue("id"), service.CapabilityRead)
		if e != nil {
			respond(w, nil, e)
			return
		}
		v, e := h.Repo.Get(r.Context(), r.PathValue("id"), "")
		respond(w, v, e)
	})))
	mux.HandleFunc("GET /api/admin/requests/{id}/history", h.withID(h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		if _, e := (service.AccessService{Repo: h.Repo}).Require(r.Context(), r.PathValue("id"), service.CapabilityRead); e != nil {
			respond(w, nil, e)
			return
		}
		v, e := h.Repo.History(r.Context(), r.PathValue("id"), "")
		respond(w, v, e)
	})))
	mux.HandleFunc("GET /api/admin/requests/{id}/photo", h.withID(h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		if _, e := (service.AccessService{Repo: h.Repo}).Require(r.Context(), r.PathValue("id"), service.CapabilityRead); e != nil {
			respond(w, nil, e)
			return
		}
		b, m, e := h.Repo.Photo(r.Context(), r.PathValue("id"), "")
		if e != nil {
			respond(w, nil, e)
			return
		}
		w.Header().Set("Content-Type", m)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	})))
	mux.HandleFunc("PATCH /api/admin/requests/{id}/status", h.withID(h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		r.Body = http.MaxBytesReader(w, r.Body, 16384)
		var in struct {
			Status  string `json:"status"`
			Comment string `json:"comment"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&in); err != nil {
			badBody(w, err)
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			badBody(w, err)
			return
		}
		svc := service.WorkflowService{Repo: h.Repo, Access: service.AccessService{Repo: h.Repo}}
		v, e := svc.SetStatus(r.Context(), r.PathValue("id"), in.Status, service.ResolveInput{FinalReport: in.Comment})
		if e == nil {
			h.Service.Notifications.Notify(r.Context(), v)
			h.Service.MAXNotifications.NotifyOwner(r.Context(), v, service.MAXEvent{Kind: service.MAXEventStatus, Text: in.Comment})
			h.Service.NotifyRequestOwnerStatusChanged(r.Context(), v, in.Comment)
		}
		respond(w, v, e)
	})))
}
