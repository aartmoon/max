package service

import (
	"context"
	"strings"
	"testing"
	"tvoydom/domain"
	"tvoydom/integration"
)

type captureRepository struct {
	RequestRepository
	saved domain.Request
}

func (r *captureRepository) Create(_ context.Context, in domain.Request) (domain.Request, error) {
	r.saved = in
	in.ID = "7"
	return in, nil
}
func TestCreateValidation(t *testing.T) {
	s := RequestService{}
	for _, in := range []CreateInput{
		{Description: " ", Address: "Москва"},
		{Description: "Течёт труба", Address: " "},
		{Description: strings.Repeat("я", 5001), Address: "Москва"},
		{Description: "Течёт труба", Address: "Москва", Kind: "UNKNOWN"},
		{Description: "Течёт труба", Address: "Москва", Photo: []byte("not an image")},
		{Description: "Течёт труба", Address: "Москва", Photo: make([]byte, MaxPhotoSize+1)},
	} {
		if _, err := s.Create(context.Background(), in); err == nil {
			t.Errorf("invalid input accepted")
		}
	}
}
func TestCreatePersistsClassificationRoutingAndText(t *testing.T) {
	repo := &captureRepository{}
	s := RequestService{Repo: repo, Classifier: RuleClassifier{}, Router: RuleRouter{}, Notifications: NotificationService{Client: integration.MockMaxClient{}}, Housing: integration.MockHousingSystemGateway{}}
	r, err := s.Create(context.Background(), CreateInput{Description: "  Не работает лифт  ", Address: "  г. Москва, дом 1  ", Kind: "APPLICATION"})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "7" || repo.saved.ProblemType != "ELEVATOR" || repo.saved.ResponsibleOrganizationID != "2" || r.Status != "CREATED" || r.Kind != "APPLICATION" {
		t.Fatalf("unexpected request: %+v", r)
	}
	if !strings.Contains(r.Text, "Не работает лифт") || r.Address != "г. Москва, дом 1" || r.Deadline.Sub(r.CreatedAt).Hours() != 72 {
		t.Fatalf("unexpected text/address/deadline: %+v", r)
	}
}
