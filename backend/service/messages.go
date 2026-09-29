package service

import (
	"context"
	"net/http"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"tvoydom/domain"
)

const MaxMessageAttachments = 5
const MaxMessageAttachmentSize = 5 << 20
const MaxMessageAttachmentsTotal = 20 << 20

type NewAttachment struct {
	Name, MIMEType string
	Data           []byte
}
type NewMessageInput struct {
	RequestID, Type, Text string
	Attachments           []NewAttachment
}
type MessageRepository interface {
	AccessRepository
	ListMessages(context.Context, string, bool) ([]domain.RequestMessage, error)
	CreateMessage(context.Context, domain.RequestMessage, []NewAttachment, string) (domain.RequestMessage, error)
	MessageAttachment(context.Context, string) (domain.MessageAttachmentContent, error)
}
type MessageService struct {
	Repo   MessageRepository
	Access AccessService
}

func (s MessageService) List(ctx context.Context, requestID string) ([]domain.RequestMessage, error) {
	if _, err := s.Access.Require(ctx, requestID, CapabilityRead); err != nil {
		return nil, err
	}
	user, _ := CurrentUser(ctx)
	return s.Repo.ListMessages(ctx, requestID, HasRole(user, "manager") || HasRole(user, "admin"))
}

func (s MessageService) Create(ctx context.Context, in NewMessageInput) (domain.RequestMessage, error) {
	user, ok := CurrentUser(ctx)
	if !ok {
		return domain.RequestMessage{}, domain.ErrUnauthorized
	}
	access, err := s.Access.Require(ctx, in.RequestID, CapabilityPublic)
	isStaff := HasRole(user, "manager") || HasRole(user, "admin")
	if isStaff {
		access, err = s.Access.Require(ctx, in.RequestID, CapabilityInternal)
	}
	if err != nil {
		return domain.RequestMessage{}, err
	}
	if access.Status == "CLOSED" {
		return domain.RequestMessage{}, domain.ErrConflict
	}
	in.Text = strings.TrimSpace(in.Text)
	if utf8.RuneCountInString(in.Text) > 5000 {
		return domain.RequestMessage{}, domain.ValidationError{Message: "Сообщение не должно превышать 5000 символов"}
	}
	if in.Text == "" && len(in.Attachments) == 0 {
		return domain.RequestMessage{}, domain.ValidationError{Message: "Введите сообщение или приложите фотографии"}
	}
	if len(in.Attachments) > MaxMessageAttachments {
		return domain.RequestMessage{}, domain.ValidationError{Message: "К сообщению можно приложить не более 5 фотографий"}
	}
	total := 0
	for i := range in.Attachments {
		a := &in.Attachments[i]
		if len(a.Data) == 0 || len(a.Data) > MaxMessageAttachmentSize {
			return domain.RequestMessage{}, domain.ValidationError{Message: "Каждое фото должно быть не больше 5 МБ"}
		}
		total += len(a.Data)
		a.MIMEType = http.DetectContentType(a.Data)
		if a.MIMEType != "image/jpeg" && a.MIMEType != "image/png" && a.MIMEType != "image/webp" {
			return domain.RequestMessage{}, domain.ValidationError{Message: "Разрешены только JPEG, PNG и WebP"}
		}
		a.Name = safeAttachmentName(a.Name)
	}
	if total > MaxMessageAttachmentsTotal {
		return domain.RequestMessage{}, domain.ValidationError{Message: "Общий размер вложений не должен превышать 20 МБ"}
	}
	role := "resident"
	if HasRole(user, "admin") {
		role = "admin"
	} else if HasRole(user, "manager") {
		role = "manager"
	}
	if role == "resident" && in.Type != "RESIDENT_PUBLIC" {
		return domain.RequestMessage{}, domain.ErrForbidden
	}
	if role != "resident" && in.Type != "ORGANIZATION_PUBLIC" && in.Type != "INTERNAL_NOTE" && in.Type != "INFO_REQUEST" {
		return domain.RequestMessage{}, domain.ValidationError{Message: "Неизвестный тип сообщения"}
	}
	awaiting := "NONE"
	if in.Type == "INFO_REQUEST" {
		awaiting = "RESIDENT"
	}
	if in.Type == "RESIDENT_PUBLIC" {
		awaiting = "ORGANIZATION"
	}
	return s.Repo.CreateMessage(ctx, domain.RequestMessage{RequestID: in.RequestID, Type: in.Type, Text: in.Text, AuthorID: user.ID, AuthorName: user.Name, AuthorRole: role}, in.Attachments, awaiting)
}

func (s MessageService) Attachment(ctx context.Context, id string) (domain.MessageAttachmentContent, error) {
	item, err := s.Repo.MessageAttachment(ctx, id)
	if err != nil {
		return item, err
	}
	if _, err := s.Access.Require(ctx, item.RequestID, CapabilityRead); err != nil {
		return domain.MessageAttachmentContent{}, err
	}
	return item, nil
}

func safeAttachmentName(value string) string {
	value = path.Base(strings.ReplaceAll(value, "\\", "/"))
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
	if strings.TrimSpace(value) == "" || value == "." {
		return "photo"
	}
	runes := []rune(value)
	if len(runes) > 150 {
		value = string(runes[:150])
	}
	return value
}
