package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"tvoydom/domain"
	"tvoydom/service"
)

// Explicitly a demo console, with no authentication or organization isolation yet.
func (h Handler) adminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/organizations", h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		v, e := h.Repo.Organizations(r.Context())
		respond(w, v, e)
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
		if service.HasRole(user, "admin") {
			v, e := h.Repo.List(r.Context(), "")
			respond(w, v, e)
			return
		}
		if user.OrganizationID == nil {
			respond(w, nil, domain.ErrForbidden)
			return
		}
		v, e := h.Repo.ListByOrganization(r.Context(), *user.OrganizationID)
		respond(w, v, e)
	}))
	mux.HandleFunc("GET /api/admin/requests/{id}", h.withID(h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		v, e := h.Repo.Get(r.Context(), r.PathValue("id"), "")
		if e == nil && !service.HasRole(user, "admin") && (user.OrganizationID == nil || r.URL.Path == "" || v.ResponsibleOrganizationID != *user.OrganizationID) {
			e = domain.ErrForbidden
		}
		respond(w, v, e)
	})))
	mux.HandleFunc("GET /api/admin/requests/{id}/history", h.withID(h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		if !service.HasRole(user, "admin") {
			v, e := h.Repo.Get(r.Context(), r.PathValue("id"), "")
			if e != nil {
				respond(w, nil, e)
				return
			}
			if user.OrganizationID == nil || v.ResponsibleOrganizationID != *user.OrganizationID {
				respond(w, nil, domain.ErrForbidden)
				return
			}
		}
		v, e := h.Repo.History(r.Context(), r.PathValue("id"), "")
		respond(w, v, e)
	})))
	mux.HandleFunc("GET /api/admin/requests/{id}/photo", h.withID(h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
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
		svc := service.AdminService{Repo: h.Repo, Notifications: h.Service.Notifications, Mailer: h.Service.Mailer}
		if !service.HasRole(user, "admin") {
			existing, e := h.Repo.Get(r.Context(), r.PathValue("id"), "")
			if e != nil {
				respond(w, nil, e)
				return
			}
			if user.OrganizationID == nil || existing.ResponsibleOrganizationID != *user.OrganizationID {
				respond(w, nil, domain.ErrForbidden)
				return
			}
		}
		v, e := svc.SetStatus(r.Context(), r.PathValue("id"), in.Status, in.Comment)
		respond(w, v, e)
	})))
}
