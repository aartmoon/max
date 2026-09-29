package controller

import (
	"net/http"

	"tvoydom/domain"
	"tvoydom/service"
)

func (h Handler) routingAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/organization-types", h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		items, err := h.Repo.OrganizationTypes(r.Context())
		respond(w, items, err)
	}))
	mux.HandleFunc("GET /api/admin/houses", h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		items, err := h.Repo.ListHouses(r.Context())
		respond(w, items, err)
	}))
	mux.HandleFunc("GET /api/admin/responsibility-rules", h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		items, err := h.Repo.ResponsibilityRules(r.Context())
		respond(w, items, err)
	}))
	mux.HandleFunc("POST /api/admin/responsibility-rules", h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		var in service.SaveRuleInput
		if err := decodeJSON(r, &in); err != nil {
			badBody(w, err)
			return
		}
		item, err := (service.RoutingAdminService{Repo: h.Repo}).SaveRule(r.Context(), "", in)
		if err != nil {
			respond(w, nil, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	}))
	mux.HandleFunc("PATCH /api/admin/responsibility-rules/{id}", h.withID(h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		var in service.SaveRuleInput
		if err := decodeJSON(r, &in); err != nil {
			badBody(w, err)
			return
		}
		item, err := (service.RoutingAdminService{Repo: h.Repo}).SaveRule(r.Context(), r.PathValue("id"), in)
		respond(w, item, err)
	})))
	mux.HandleFunc("PATCH /api/admin/organizations/{id}", h.withID(h.requireRole("admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		var in service.SaveOrganizationInput
		if err := decodeJSON(r, &in); err != nil {
			badBody(w, err)
			return
		}
		item, err := (service.RoutingAdminService{Repo: h.Repo}).SaveOrganization(r.Context(), r.PathValue("id"), in)
		respond(w, item, err)
	})))
	mux.HandleFunc("GET /api/organization/staff", h.requireRole("manager", "admin")(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		organizationID := ""
		if user.OrganizationID != nil {
			organizationID = *user.OrganizationID
		}
		if service.HasRole(user, "admin") && r.URL.Query().Get("organizationId") != "" {
			organizationID = r.URL.Query().Get("organizationId")
		}
		if organizationID == "" {
			respond(w, nil, domain.ErrForbidden)
			return
		}
		items, err := h.Repo.UsersByOrganization(r.Context(), organizationID)
		respond(w, items, err)
	}))
}
