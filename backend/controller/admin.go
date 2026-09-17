package controller

import (
	"encoding/json"
	"io"
	"net/http"
	"tvoydom/service"
)

// Explicitly a demo console, with no authentication or organization isolation yet.
func (h Handler) adminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/organizations", func(w http.ResponseWriter, r *http.Request) {
		v, e := h.Repo.Organizations(r.Context())
		respond(w, v, e)
	})
	mux.HandleFunc("GET /api/admin/requests", func(w http.ResponseWriter, r *http.Request) { v, e := h.Repo.List(r.Context(), ""); respond(w, v, e) })
	mux.HandleFunc("GET /api/admin/requests/{id}", h.withID(func(w http.ResponseWriter, r *http.Request) {
		v, e := h.Repo.Get(r.Context(), r.PathValue("id"), "")
		respond(w, v, e)
	}))
	mux.HandleFunc("GET /api/admin/requests/{id}/history", h.withID(func(w http.ResponseWriter, r *http.Request) {
		v, e := h.Repo.History(r.Context(), r.PathValue("id"), "")
		respond(w, v, e)
	}))
	mux.HandleFunc("GET /api/admin/requests/{id}/photo", h.withID(func(w http.ResponseWriter, r *http.Request) {
		b, m, e := h.Repo.Photo(r.Context(), r.PathValue("id"), "")
		if e != nil {
			respond(w, nil, e)
			return
		}
		w.Header().Set("Content-Type", m)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	}))
	mux.HandleFunc("PATCH /api/admin/requests/{id}/status", h.withID(func(w http.ResponseWriter, r *http.Request) {
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
		svc := service.AdminService{Repo: h.Repo, Notifications: h.Service.Notifications}
		v, e := svc.SetStatus(r.Context(), r.PathValue("id"), in.Status, in.Comment)
		respond(w, v, e)
	}))
}
