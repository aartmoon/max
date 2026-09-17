package controller

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"tvoydom/domain"
	"tvoydom/repository"
	"tvoydom/service"
)

type Handler struct {
	Service           service.RequestService
	Houses            service.HouseService
	Addresses         service.AddressProvider
	Repo              repository.Postgres
	MockStatusEnabled bool
}

func (h Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	h.adminRoutes(mux)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := h.Repo.Pool.Ping(r.Context()); err != nil {
			writeJSON(w, 503, map[string]string{"error": "База данных недоступна"})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/requests", h.create)
	mux.HandleFunc("GET /api/addresses/search", h.searchAddresses)
	mux.HandleFunc("GET /api/addresses/{fiasGuid}", func(w http.ResponseWriter, r *http.Request) {
		v, e := h.Addresses.GetByGUID(r.Context(), r.PathValue("fiasGuid"))
		respond(w, v, e)
	})
	mux.HandleFunc("POST /api/houses/resolve", h.resolveHouse)
	mux.HandleFunc("GET /api/houses/{id}", h.withID(func(w http.ResponseWriter, r *http.Request) {
		v, e := h.Repo.GetHouse(r.Context(), r.PathValue("id"))
		if e != nil {
			respond(w, nil, e)
			return
		}
		writeJSON(w, 200, houseIdentityResponse(v))
	}))
	mux.HandleFunc("GET /api/requests", func(w http.ResponseWriter, r *http.Request) { v, e := h.Service.List(r.Context()); respond(w, v, e) })
	mux.HandleFunc("GET /api/requests/{id}", h.withID(func(w http.ResponseWriter, r *http.Request) {
		v, e := h.Service.Get(r.Context(), r.PathValue("id"))
		respond(w, v, e)
	}))
	mux.HandleFunc("GET /api/requests/{id}/history", h.withID(func(w http.ResponseWriter, r *http.Request) {
		v, e := h.Service.History(r.Context(), r.PathValue("id"))
		respond(w, v, e)
	}))
	mux.HandleFunc("GET /api/requests/{id}/photo", h.withID(func(w http.ResponseWriter, r *http.Request) {
		b, m, e := h.Repo.Photo(r.Context(), r.PathValue("id"), service.DemoUserID)
		if e != nil {
			respond(w, nil, e)
			return
		}
		w.Header().Set("Content-Type", m)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	}))
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]bool{"mockStatusEnabled": h.MockStatusEnabled})
	})
	if h.MockStatusEnabled {
		mux.HandleFunc("POST /api/requests/{id}/mock-next-status", h.withID(func(w http.ResponseWriter, r *http.Request) {
			v, e := h.Service.Next(r.Context(), r.PathValue("id"), r.URL.Query().Get("reject") == "true")
			respond(w, v, e)
		}))
	}
	return mux
}

func (h Handler) searchAddresses(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 50 {
			writeJSON(w, 400, map[string]any{"error": map[string]string{"code": "invalid_request", "message": "Некорректный limit"}})
			return
		}
		limit = parsed
	}
	items, err := h.Addresses.Search(r.Context(), r.URL.Query().Get("q"), limit)
	if err != nil {
		respond(w, nil, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (h Handler) resolveHouse(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	var in struct {
		FIASGUID string `json:"fiasGuid"`
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
	v, e := h.Houses.Resolve(r.Context(), in.FIASGUID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	writeJSON(w, 200, houseIdentityResponse(v))
}

func houseIdentityResponse(h domain.House) map[string]any {
	return map[string]any{
		"id":              h.ID,
		"fiasGuid":        h.FIASGUID,
		"address":         h.Address,
		"cadastralNumber": h.CadastralNumber,
	}
}

func (h Handler) withID(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if e != nil || id <= 0 {
			writeJSON(w, 400, map[string]string{"error": "Некорректный номер обращения"})
			return
		}
		fn(w, r)
	}
}
func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	var in service.CreateInput
	media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if media == "multipart/form-data" {
		if err := r.ParseMultipartForm(6 << 20); err != nil {
			badBody(w, err)
			return
		}
		defer r.MultipartForm.RemoveAll()
		in.Description = r.FormValue("description")
		in.Address = r.FormValue("address")
		in.Kind = r.FormValue("kind")
		f, _, err := r.FormFile("photo")
		if err == nil {
			defer f.Close()
			in.Photo, err = io.ReadAll(io.LimitReader(f, service.MaxPhotoSize+1))
			if err != nil {
				badBody(w, err)
				return
			}
		} else if !errors.Is(err, http.ErrMissingFile) {
			badBody(w, err)
			return
		}
	} else if media == "application/json" {
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if err := d.Decode(&in); err != nil {
			badBody(w, err)
			return
		}
		if err := d.Decode(&struct{}{}); err != io.EOF {
			badBody(w, err)
			return
		}
	} else {
		writeJSON(w, 415, map[string]string{"error": "Используйте JSON или multipart/form-data"})
		return
	}
	v, e := h.Service.Create(r.Context(), in)
	if e != nil {
		respond(w, nil, e)
		return
	}
	writeJSON(w, 201, v)
}
func badBody(w http.ResponseWriter, err error) {
	var max *http.MaxBytesError
	if errors.As(err, &max) {
		writeJSON(w, 413, map[string]string{"error": "Размер запроса превышает 6 МБ"})
		return
	}
	writeJSON(w, 400, map[string]string{"error": "Не удалось прочитать данные обращения"})
}
func respond(w http.ResponseWriter, v any, err error) {
	if err == nil {
		writeJSON(w, 200, v)
		return
	}
	code := 500
	message := "Не удалось выполнить запрос. Попробуйте ещё раз."
	var validation domain.ValidationError
	switch {
	case errors.As(err, &validation):
		code = 400
		message = err.Error()
	case errors.Is(err, domain.ErrNotFound):
		code = 404
		message = err.Error()
	case errors.Is(err, domain.ErrConflict):
		code = 409
		message = err.Error()
	case errors.Is(err, domain.ErrInvalidFIASGUID):
		code = 400
		message = err.Error()
	case errors.Is(err, domain.ErrInvalidHouse):
		code = 400
		message = err.Error()
	default:
		slog.Error("request failed", "error", err)
	}
	writeJSON(w, code, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
