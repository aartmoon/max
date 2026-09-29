package controller

import (
	"io"
	"mime"
	"net/http"

	"tvoydom/domain"
	"tvoydom/service"
)

func (h Handler) messageRoutes(mux *http.ServeMux) {
	list := h.withID(h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		items, err := (service.MessageService{Repo: h.Repo, Access: service.AccessService{Repo: h.Repo}}).List(r.Context(), r.PathValue("id"))
		respond(w, items, err)
	}))
	create := h.withID(h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) { h.createMessage(w, r, user) }))
	mux.HandleFunc("GET /api/requests/{id}/messages", list)
	mux.HandleFunc("POST /api/requests/{id}/messages", create)
	mux.HandleFunc("GET /api/organization/requests/{id}/messages", list)
	mux.HandleFunc("POST /api/organization/requests/{id}/messages", create)
	mux.HandleFunc("GET /api/request-attachments/{id}", h.withID(h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		item, err := (service.MessageService{Repo: h.Repo, Access: service.AccessService{Repo: h.Repo}}).Attachment(r.Context(), r.PathValue("id"))
		if err != nil {
			respond(w, nil, err)
			return
		}
		w.Header().Set("Content-Type", item.MIMEType)
		w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": item.Name}))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(item.Data)
	})))
}

func (h Handler) createMessage(w http.ResponseWriter, r *http.Request, user domain.User) {
	r.Body = http.MaxBytesReader(w, r.Body, 21<<20)
	if err := r.ParseMultipartForm(21 << 20); err != nil {
		badBody(w, err)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	in := service.NewMessageInput{RequestID: r.PathValue("id"), Type: r.FormValue("type"), Text: r.FormValue("text")}
	if r.MultipartForm != nil {
		for _, header := range r.MultipartForm.File["attachments"] {
			file, err := header.Open()
			if err != nil {
				badBody(w, err)
				return
			}
			data, readErr := io.ReadAll(io.LimitReader(file, service.MaxMessageAttachmentSize+1))
			_ = file.Close()
			if readErr != nil {
				badBody(w, readErr)
				return
			}
			in.Attachments = append(in.Attachments, service.NewAttachment{Name: header.Filename, Data: data})
		}
	}
	message, err := (service.MessageService{Repo: h.Repo, Access: service.AccessService{Repo: h.Repo}}).Create(r.Context(), in)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if message.Type != "INTERNAL_NOTE" {
		if request, getErr := h.Repo.Get(r.Context(), in.RequestID, ""); getErr == nil {
			h.Service.Notifications.Notify(r.Context(), request)
			if service.HasRole(user, "resident") {
				h.Service.MAXNotifications.NotifyManagers(r.Context(), request, service.MAXEvent{Kind: service.MAXEventMessage, Text: message.Text})
				h.Service.NotifyManagers(r.Context(), request)
			} else {
				h.Service.MAXNotifications.NotifyOwner(r.Context(), request, service.MAXEvent{Kind: service.MAXEventMessage, Text: message.Text})
				h.Service.NotifyRequestOwnerStatusChanged(r.Context(), request, "Новое сообщение по заявке")
			}
		}
	}
	writeJSON(w, http.StatusCreated, message)
}
