package controller

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"tvoydom/domain"
	"tvoydom/repository"
	"tvoydom/service"
	"unicode/utf8"
)

type Handler struct {
	Service           service.RequestService
	Auth              service.AuthService
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
	mux.HandleFunc("POST /api/auth/request-code", h.requestAuthCode)
	mux.HandleFunc("POST /api/auth/verify-code", h.verifyAuthCode)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/me", h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		writeJSON(w, 200, user)
	}))
	mux.HandleFunc("GET /api/me/apartments", h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		v, e := h.Repo.UserApartments(r.Context(), user.ID)
		respond(w, v, e)
	}))
	mux.HandleFunc("POST /api/me/apartments", h.requireUser(h.createApartment))
	mux.HandleFunc("PATCH /api/me/apartments/{id}/default", h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		v, e := h.Repo.SetDefaultApartment(r.Context(), user.ID, r.PathValue("id"))
		respond(w, v, e)
	}))
	mux.HandleFunc("DELETE /api/me/apartments/{id}", h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		e := h.Repo.DeleteUserApartment(r.Context(), user.ID, r.PathValue("id"))
		respond(w, map[string]bool{"ok": true}, e)
	}))
	mux.HandleFunc("POST /api/requests", h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		h.create(w, r.WithContext(service.WithCurrentUser(r.Context(), user)))
	}))
	mux.HandleFunc("GET /api/addresses/search", h.searchAddresses)
	mux.HandleFunc("GET /api/addresses/{objectId}", func(w http.ResponseWriter, r *http.Request) {
		objectID, e := parsePositiveID(r.PathValue("objectId"))
		if e != nil {
			respond(w, nil, e)
			return
		}
		v, e := h.Addresses.GetAddress(r.Context(), objectID)
		respond(w, v, e)
	})
	mux.HandleFunc("POST /api/houses/resolve", h.resolveHouse)
	mux.HandleFunc("GET /api/houses/{id}", h.withID(func(w http.ResponseWriter, r *http.Request) {
		v, e := h.Repo.GetHouse(r.Context(), r.PathValue("id"))
		if e != nil {
			respond(w, nil, e)
			return
		}
		writeJSON(w, 200, houseResponse(v))
	}))
	mux.HandleFunc("GET /api/requests", h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		v, e := h.Service.List(service.WithCurrentUser(r.Context(), user))
		respond(w, v, e)
	}))
	mux.HandleFunc("GET /api/requests/{id}", h.requireUserID(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		v, e := h.Service.Get(r.Context(), r.PathValue("id"))
		respond(w, v, e)
	}))
	mux.HandleFunc("GET /api/requests/{id}/history", h.requireUserID(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		v, e := h.Service.History(r.Context(), r.PathValue("id"))
		respond(w, v, e)
	}))
	mux.HandleFunc("GET /api/requests/{id}/photo", h.requireUserID(func(w http.ResponseWriter, r *http.Request, user domain.User) {
		b, m, e := h.Repo.Photo(r.Context(), r.PathValue("id"), user.ID)
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
	return mux
}

func (h Handler) requestAuthCode(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	var in struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &in); err != nil {
		badBody(w, err)
		return
	}
	if err := h.Auth.RequestCode(r.Context(), in.Email); err != nil {
		respond(w, nil, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (h Handler) verifyAuthCode(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	var in struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := decodeJSON(r, &in); err != nil {
		badBody(w, err)
		return
	}
	session, err := h.Auth.VerifyCode(r.Context(), in.Email, in.Code)
	if err != nil {
		respond(w, nil, err)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: service.SessionCookieName, Value: session.Token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 30 * 24 * 60 * 60})
	writeJSON(w, 200, session.User)
}

func (h Handler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(service.SessionCookieName)
	if err == nil {
		_ = h.Auth.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: service.SessionCookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func decodeJSON(r *http.Request, v any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return err
	}
	return nil
}

func (h Handler) currentUser(r *http.Request) (domain.User, error) {
	cookie, err := r.Cookie(service.SessionCookieName)
	if err != nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	return h.Auth.UserByToken(r.Context(), cookie.Value)
}

func (h Handler) requireUser(fn func(http.ResponseWriter, *http.Request, domain.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := h.currentUser(r)
		if err != nil {
			respond(w, nil, err)
			return
		}
		fn(w, r.WithContext(service.WithCurrentUser(r.Context(), user)), user)
	}
}

func (h Handler) requireUserID(fn func(http.ResponseWriter, *http.Request, domain.User)) http.HandlerFunc {
	return h.withID(h.requireUser(fn))
}

func (h Handler) requireRole(roles ...string) func(func(http.ResponseWriter, *http.Request, domain.User)) http.HandlerFunc {
	return func(fn func(http.ResponseWriter, *http.Request, domain.User)) http.HandlerFunc {
		return h.requireUser(func(w http.ResponseWriter, r *http.Request, user domain.User) {
			for _, role := range roles {
				if service.HasRole(user, role) {
					fn(w, r, user)
					return
				}
			}
			respond(w, nil, domain.ErrForbidden)
		})
	}
}

func (h Handler) searchAddresses(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(query) > 300 {
		writeJSON(w, 400, map[string]string{"error": "Поисковый запрос слишком длинный"})
		return
	}
	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 50 {
			writeJSON(w, 400, map[string]string{"error": "Некорректный limit"})
			return
		}
		limit = parsed
	}
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	if kind != "" && !validAddressKind(kind) {
		writeJSON(w, 400, map[string]string{"error": "Некорректный тип адресного объекта"})
		return
	}
	var parent *int64
	if raw := strings.TrimSpace(r.URL.Query().Get("parentObjectId")); raw != "" {
		value, err := parsePositiveID(raw)
		if err != nil {
			respond(w, nil, err)
			return
		}
		parent = &value
	}
	if query == "" && parent == nil {
		writeJSON(w, 400, map[string]string{"error": "Укажите поисковый запрос или родительский объект"})
		return
	}
	items, err := h.Addresses.Search(r.Context(), domain.AddressSearch{Query: query, Kind: kind, ParentObjectID: parent, Limit: limit})
	if err != nil {
		respond(w, nil, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}

func (h Handler) resolveHouse(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	var in struct {
		ObjectID string `json:"objectId"`
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
	v, e := h.Houses.Resolve(r.Context(), in.ObjectID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	writeJSON(w, 200, houseResponse(v))
}

func (h Handler) createApartment(w http.ResponseWriter, r *http.Request, user domain.User) {
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	var in struct {
		HouseObjectID     string `json:"houseObjectId"`
		ApartmentObjectID string `json:"apartmentObjectId"`
		Label             string `json:"label"`
		IsDefault         bool   `json:"isDefault"`
	}
	if err := decodeJSON(r, &in); err != nil {
		badBody(w, err)
		return
	}
	houseObjectID, err := parsePositiveID(in.HouseObjectID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	house, err := h.Addresses.GetAddress(r.Context(), houseObjectID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if house.ObjectKind != "house" || !house.IsActive {
		respond(w, nil, domain.ErrInvalidHouse)
		return
	}
	address := house.FullAddress
	a := domain.UserApartment{UserID: user.ID, HouseObjectID: house.ObjectID, HouseObjectGUID: house.ObjectGUID, Address: address, Label: strings.TrimSpace(in.Label), IsDefault: in.IsDefault}
	if strings.TrimSpace(in.ApartmentObjectID) != "" {
		apartmentObjectID, err := parsePositiveID(in.ApartmentObjectID)
		if err != nil {
			respond(w, nil, domain.ErrInvalidApartment)
			return
		}
		apartment, err := h.Addresses.GetAddress(r.Context(), apartmentObjectID)
		if err != nil {
			respond(w, nil, err)
			return
		}
		if apartment.ObjectKind != "apartment" || !apartment.IsActive || apartment.ParentObjectID != house.ObjectID {
			respond(w, nil, domain.ErrInvalidApartment)
			return
		}
		a.ApartmentObjectID = apartment.ObjectID
		a.ApartmentObjectGUID = apartment.ObjectGUID
		a.Address = apartment.FullAddress
	}
	v, e := h.Repo.CreateUserApartment(r.Context(), a)
	respond(w, v, e)
}

func houseResponse(h domain.House) map[string]any {
	return map[string]any{
		"id":              h.ID,
		"garObjectId":     h.GARObjectID,
		"objectGuid":      h.ObjectGUID,
		"address":         h.Address,
		"cadastralNumber": h.CadastralNumber,
		"totalArea":       h.TotalArea,
		"livingArea":      h.LivingArea,
		"floors":          h.Floors,
		"entrances":       h.Entrances,
		"apartments":      h.Apartments,
		"yearBuilt":       h.YearBuilt,
		"organization":    h.Organization,
		"manager":         h.Manager,
		"contact":         h.Contact,
		"dataSource":      h.DataSource,
		"dataUpdatedAt":   h.DataUpdatedAt,
		"stale":           h.Stale,
		"characteristics": h.Characteristics,
		"management":      h.Management,
		"dataSources":     h.DataSources,
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
		in.HouseObjectID = r.FormValue("houseObjectId")
		in.ApartmentObjectID = r.FormValue("apartmentObjectId")
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
	case errors.Is(err, domain.ErrHouseProfileNotFound):
		code = 404
		message = err.Error()
	case errors.Is(err, domain.ErrHouseProfileUnavailable):
		code = 503
		message = err.Error()
	case errors.Is(err, domain.ErrConflict):
		code = 409
		message = err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		code = 401
		message = err.Error()
	case errors.Is(err, domain.ErrForbidden):
		code = 403
		message = err.Error()
	case errors.Is(err, domain.ErrInvalidAddressID):
		code = 400
		message = err.Error()
	case errors.Is(err, domain.ErrInvalidHouse):
		code = 400
		message = err.Error()
	case errors.Is(err, domain.ErrInvalidApartment):
		code = 400
		message = err.Error()
	default:
		slog.Error("request failed", "error", err)
	}
	writeJSON(w, code, map[string]string{"error": message})
}

func parsePositiveID(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		return 0, domain.ErrInvalidAddressID
	}
	return value, nil
}

func validAddressKind(kind string) bool {
	switch kind {
	case "address_object", "house", "apartment", "room", "carplace", "stead":
		return true
	default:
		return false
	}
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
