package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"tvoydom/domain"
	"tvoydom/integration"
	"unicode/utf8"
)

const DemoUserID = "1"
const MaxPhotoSize = 5 << 20

type RequestRepository interface {
	Create(context.Context, domain.Request) (domain.Request, error)
	List(context.Context, string) ([]domain.Request, error)
	Get(context.Context, string, string) (domain.Request, error)
	History(context.Context, string, string) ([]domain.RequestStatusHistory, error)
	Transition(context.Context, string, string, func(string) (string, error)) (domain.Request, error)
}
type NotificationService struct{ Client integration.MaxClient }

func (n NotificationService) Notify(ctx context.Context, r domain.Request) {
	if err := n.Client.Notify(ctx, r.UserID, "Обращение №"+r.ID+": "+r.Status); err != nil {
		slog.Error("notification failed", "requestId", r.ID, "error", err)
	}
}

type RequestService struct {
	Repo          RequestRepository
	Addresses     AddressProvider
	Houses        HouseRepository
	Classifier    Classifier
	Router        Router
	Notifications NotificationService
	Mailer        Mailer
	Housing       integration.HousingSystemGateway
}
type CreateInput struct {
	Description       string `json:"description"`
	HouseObjectID     string `json:"houseObjectId"`
	ApartmentObjectID string `json:"apartmentObjectId"`
	Kind              string `json:"kind"`
	Photo             []byte `json:"-"`
}

func (s RequestService) Create(ctx context.Context, in CreateInput) (domain.Request, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return domain.Request{}, domain.ErrUnauthorized
	}
	in.Description = strings.TrimSpace(in.Description)
	in.HouseObjectID = strings.TrimSpace(in.HouseObjectID)
	in.ApartmentObjectID = strings.TrimSpace(in.ApartmentObjectID)
	if n := utf8.RuneCountInString(in.Description); n < 5 || n > 5000 {
		return domain.Request{}, domain.ValidationError{Message: "Описание должно содержать от 5 до 5000 символов"}
	}
	if in.Kind == "" {
		in.Kind = "APPLICATION"
	}
	if in.Kind != "PROBLEM" && in.Kind != "APPLICATION" && in.Kind != "QUESTION" && in.Kind != "EMERGENCY" && in.Kind != "COMPLAINT" {
		return domain.Request{}, domain.ValidationError{Message: "Неизвестный тип обращения"}
	}
	photoType := ""
	if len(in.Photo) > 0 {
		photoType = http.DetectContentType(in.Photo)
		if len(in.Photo) > MaxPhotoSize || (photoType != "image/jpeg" && photoType != "image/png" && photoType != "image/webp") {
			return domain.Request{}, domain.ValidationError{Message: "Фото: JPEG, PNG или WebP, не более 5 МБ"}
		}
	}
	houseObjectID, err := parseObjectID(in.HouseObjectID)
	if err != nil {
		return domain.Request{}, err
	}
	houseAddress, err := s.Addresses.GetAddress(ctx, houseObjectID)
	if err != nil {
		return domain.Request{}, err
	}
	if houseAddress.ObjectKind != "house" || !houseAddress.IsActive {
		return domain.Request{}, domain.ErrInvalidHouse
	}
	address := houseAddress.FullAddress
	var apartment domain.AddressInfo
	if in.ApartmentObjectID != "" {
		apartmentObjectID, err := parseObjectID(in.ApartmentObjectID)
		if err != nil {
			return domain.Request{}, domain.ErrInvalidApartment
		}
		apartment, err = s.Addresses.GetAddress(ctx, apartmentObjectID)
		if err != nil {
			return domain.Request{}, err
		}
		if apartment.ObjectKind != "apartment" || !apartment.IsActive || apartment.ParentObjectID != houseAddress.ObjectID {
			return domain.Request{}, domain.ErrInvalidApartment
		}
		address = apartment.FullAddress
	}
	house, err := s.Houses.ResolveHouse(ctx, houseAddress)
	if err != nil {
		return domain.Request{}, err
	}
	category := s.Classifier.Classify(in.Description)
	org := s.Router.Route(category)
	now := time.Now().UTC()
	r, err := s.Repo.Create(ctx, domain.Request{
		UserID: user.ID, HouseID: house.ID, Address: address, Description: in.Description, Kind: in.Kind,
		ProblemType: category, ResponsibleOrganizationID: org.ID, ResponsibleOrganization: org.Name,
		Status: "CREATED", Deadline: now.AddDate(0, 0, 3), CreatedAt: now,
		Photo: in.Photo, PhotoType: photoType, HasPhoto: len(in.Photo) > 0,
		HouseObjectID: houseAddress.ObjectID, HouseObjectGUID: houseAddress.ObjectGUID,
		ApartmentObjectID: apartment.ObjectID, ApartmentObjectGUID: apartment.ObjectGUID,
		Text: fmt.Sprintf("В %s\nАдрес: %s\n\n%s\n\nПрошу рассмотреть обращение и сообщить о результате.", org.Name, address, in.Description),
	})
	if err == nil {
		s.Notifications.Notify(ctx, r)
		s.NotifyManagers(ctx, r)
	}
	return r, err
}
func (s RequestService) List(ctx context.Context) ([]domain.Request, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	return s.Repo.List(ctx, user.ID)
}
func (s RequestService) Get(ctx context.Context, id string) (domain.Request, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return domain.Request{}, domain.ErrUnauthorized
	}
	return s.Repo.Get(ctx, id, user.ID)
}
func (s RequestService) History(ctx context.Context, id string) ([]domain.RequestStatusHistory, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	return s.Repo.History(ctx, id, user.ID)
}
func (s RequestService) Next(ctx context.Context, id string, reject bool) (domain.Request, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return domain.Request{}, domain.ErrUnauthorized
	}
	r, err := s.Repo.Transition(ctx, id, user.ID, func(status string) (string, error) {
		if reject {
			if status == "RESOLVED" || status == "REJECTED" {
				return "", domain.ErrConflict
			}
			return "REJECTED", nil
		}
		return NextStatus(status)
	})
	if err != nil {
		return r, err
	}
	// The demo adapter cannot fail. A real adapter needs durable delivery and retries.
	if r.Status == "SENT" {
		if err := s.Housing.Submit(ctx, r); err != nil {
			slog.Error("housing delivery failed", "requestId", r.ID, "error", err)
		}
	}
	s.Notifications.Notify(ctx, r)
	s.NotifyRequestOwnerStatusChanged(ctx, r, "")
	return r, nil
}

type managerEmailRepository interface {
	ManagerEmailsForOrganization(context.Context, string) ([]string, error)
}

type requestOwnerEmailRepository interface {
	RequestOwnerEmail(context.Context, string) (string, error)
}

func notifyRequestOwnerStatusChanged(ctx context.Context, repo any, mailer Mailer, r domain.Request, comment string) {
	if mailer == nil {
		return
	}
	emailRepo, ok := repo.(requestOwnerEmailRepository)
	if !ok {
		return
	}
	email, err := emailRepo.RequestOwnerEmail(ctx, r.ID)
	if err != nil {
		slog.Error("request owner email lookup failed", "requestId", r.ID, "error", err)
		return
	}
	if strings.TrimSpace(email) == "" {
		return
	}
	if err := mailer.SendRequestStatusChanged(ctx, email, r, comment); err != nil {
		slog.Error("request status email failed", "requestId", r.ID, "email", email, "error", err)
	}
}

func (s RequestService) NotifyRequestOwnerStatusChanged(ctx context.Context, r domain.Request, comment string) {
	notifyRequestOwnerStatusChanged(ctx, s.Repo, s.Mailer, r, comment)
}

func (s RequestService) NotifyManagers(ctx context.Context, r domain.Request) {
	if s.Mailer == nil {
		return
	}
	repo, ok := s.Repo.(managerEmailRepository)
	if !ok {
		return
	}
	emails, err := repo.ManagerEmailsForOrganization(ctx, r.ResponsibleOrganizationID)
	if err != nil {
		slog.Error("manager lookup failed", "requestId", r.ID, "error", err)
		return
	}
	for _, email := range emails {
		if err := s.Mailer.SendNewRequest(ctx, email, r); err != nil {
			slog.Error("manager email failed", "requestId", r.ID, "email", email, "error", err)
		}
	}
}
