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
	Classifier    Classifier
	Router        Router
	Notifications NotificationService
	Housing       integration.HousingSystemGateway
}
type CreateInput struct {
	Description string `json:"description"`
	Address     string `json:"address"`
	Kind        string `json:"kind"`
	Photo       []byte `json:"-"`
}

func (s RequestService) Create(ctx context.Context, in CreateInput) (domain.Request, error) {
	in.Description = strings.TrimSpace(in.Description)
	in.Address = strings.TrimSpace(in.Address)
	if n := utf8.RuneCountInString(in.Description); n < 5 || n > 5000 {
		return domain.Request{}, domain.ValidationError{Message: "Описание должно содержать от 5 до 5000 символов"}
	}
	if n := utf8.RuneCountInString(in.Address); n < 5 || n > 300 {
		return domain.Request{}, domain.ValidationError{Message: "Адрес должен содержать от 5 до 300 символов"}
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
	category := s.Classifier.Classify(in.Description)
	org := s.Router.Route(category)
	now := time.Now().UTC()
	r, err := s.Repo.Create(ctx, domain.Request{UserID: DemoUserID, Address: in.Address, Description: in.Description, Kind: in.Kind, ProblemType: category, ResponsibleOrganizationID: org.ID, ResponsibleOrganization: org.Name, Status: "CREATED", Deadline: now.AddDate(0, 0, 3), CreatedAt: now, Photo: in.Photo, PhotoType: photoType, HasPhoto: len(in.Photo) > 0, Text: fmt.Sprintf("В %s\nАдрес: %s\n\n%s\n\nПрошу рассмотреть обращение и сообщить о результате.", org.Name, in.Address, in.Description)})
	if err == nil {
		s.Notifications.Notify(ctx, r)
	}
	return r, err
}
func (s RequestService) List(ctx context.Context) ([]domain.Request, error) {
	return s.Repo.List(ctx, DemoUserID)
}
func (s RequestService) Get(ctx context.Context, id string) (domain.Request, error) {
	return s.Repo.Get(ctx, id, DemoUserID)
}
func (s RequestService) History(ctx context.Context, id string) ([]domain.RequestStatusHistory, error) {
	return s.Repo.History(ctx, id, DemoUserID)
}
func (s RequestService) Next(ctx context.Context, id string, reject bool) (domain.Request, error) {
	r, err := s.Repo.Transition(ctx, id, DemoUserID, func(status string) (string, error) {
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
	return r, nil
}
