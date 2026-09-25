package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tvoydom/domain"
	"tvoydom/integration"
)

type captureRepository struct {
	RequestRepository
	saved      domain.Request
	statusFrom string
}

func (r *captureRepository) Create(_ context.Context, in domain.Request) (domain.Request, error) {
	r.saved = in
	in.ID = "7"
	return in, nil
}

func (r *captureRepository) Transition(_ context.Context, id, user string, next func(string) (string, error)) (domain.Request, error) {
	from := r.statusFrom
	if from == "" {
		from = "CREATED"
	}
	status, err := next(from)
	return domain.Request{ID: id, UserID: user, Status: status}, err
}

func (r *captureRepository) RequestOwnerEmail(_ context.Context, id string) (string, error) {
	return "resident@example.com", nil
}

func TestCreateRejectsMissingOrInvalidGARHouse(t *testing.T) {
	addresses := fakeAddressProvider{items: map[int64]domain.AddressInfo{
		10: {ObjectID: "10", ObjectKind: "apartment", IsActive: true},
		11: {ObjectID: "11", ObjectKind: "house", IsActive: false},
	}}
	for _, houseID := range []string{"", "text", "10", "11", "999"} {
		svc := requestServiceForTest(&captureRepository{}, addresses)
		_, err := svc.Create(authContext(), CreateInput{Description: "Течёт труба", Kind: "PROBLEM", HouseObjectID: houseID})
		if err == nil {
			t.Errorf("house %q was accepted", houseID)
		}
	}
}

func TestCreateRejectsApartmentFromAnotherHouse(t *testing.T) {
	addresses := fakeAddressProvider{items: map[int64]domain.AddressInfo{
		10: {ObjectID: "10", ObjectKind: "house", ObjectGUID: "10000000-0000-0000-0000-000000000010", FullAddress: "г. Москва, д. 1", IsActive: true},
		20: {ObjectID: "20", ParentObjectID: "11", ObjectKind: "apartment", ObjectGUID: "20000000-0000-0000-0000-000000000020", FullAddress: "г. Москва, д. 2, кв. 1", IsActive: true},
	}}
	svc := requestServiceForTest(&captureRepository{}, addresses)
	_, err := svc.Create(authContext(), CreateInput{Description: "Течёт труба", Kind: "PROBLEM", HouseObjectID: "10", ApartmentObjectID: "20"})
	if !errors.Is(err, domain.ErrInvalidApartment) {
		t.Fatalf("got %v", err)
	}
}

func TestCreatePersistsValidatedGARSelectionAndAddressSnapshot(t *testing.T) {
	repo := &captureRepository{}
	addresses := fakeAddressProvider{items: map[int64]domain.AddressInfo{
		10: {ObjectID: "10", ObjectGUID: "10000000-0000-0000-0000-000000000010", ObjectKind: "house", DisplayName: "д. 1", FullAddress: "г. Москва, ул. Тверская, д. 1", IsActive: true},
		20: {ObjectID: "20", ObjectGUID: "20000000-0000-0000-0000-000000000020", ParentObjectID: "10", ObjectKind: "apartment", DisplayName: "кв. 5", FullAddress: "г. Москва, ул. Тверская, д. 1, кв. 5", IsActive: true},
	}}
	svc := requestServiceForTest(repo, addresses)
	r, err := svc.Create(authContext(), CreateInput{Description: "  Не работает лифт  ", Kind: "APPLICATION", HouseObjectID: "10", ApartmentObjectID: "20"})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "7" || repo.saved.UserID != "99" || repo.saved.ProblemType != "ELEVATOR" || repo.saved.HouseID != "42" || r.Status != "CREATED" {
		t.Fatalf("unexpected request: %+v", r)
	}
	if r.Address != addresses.items[20].FullAddress || r.HouseObjectID != "10" || r.ApartmentObjectID != "20" {
		t.Fatalf("GAR selection was not persisted: %+v", r)
	}
	if !strings.Contains(r.Text, r.Address) || r.Deadline.Sub(r.CreatedAt).Hours() != 72 {
		t.Fatalf("unexpected text/deadline: %+v", r)
	}
}

func TestCreateUsesHouseAddressWhenApartmentOmitted(t *testing.T) {
	repo := &captureRepository{}
	addresses := fakeAddressProvider{items: map[int64]domain.AddressInfo{
		10: {ObjectID: "10", ObjectGUID: "10000000-0000-0000-0000-000000000010", ObjectKind: "house", FullAddress: "г. Москва, ул. Арбат, д. 12", IsActive: true},
	}}
	svc := requestServiceForTest(repo, addresses)
	r, err := svc.Create(authContext(), CreateInput{Description: "Течёт труба", HouseObjectID: "10"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Address != addresses.items[10].FullAddress || r.ApartmentObjectID != "" {
		t.Fatalf("unexpected request: %+v", r)
	}
}

func TestCreateValidatesDescriptionKindAndPhoto(t *testing.T) {
	addresses := fakeAddressProvider{items: map[int64]domain.AddressInfo{10: {ObjectID: "10", ObjectKind: "house", IsActive: true}}}
	for _, in := range []CreateInput{
		{Description: " ", HouseObjectID: "10"},
		{Description: strings.Repeat("я", 5001), HouseObjectID: "10"},
		{Description: "Течёт труба", HouseObjectID: "10", Kind: "UNKNOWN"},
		{Description: "Течёт труба", HouseObjectID: "10", Photo: []byte("not an image")},
		{Description: "Течёт труба", HouseObjectID: "10", Photo: make([]byte, MaxPhotoSize+1)},
	} {
		if _, err := requestServiceForTest(&captureRepository{}, addresses).Create(authContext(), in); err == nil {
			t.Errorf("invalid input accepted: %+v", in)
		}
	}
}

func TestNextEmailsRequestOwnerAfterStatusChange(t *testing.T) {
	repo := &captureRepository{}
	mailer := &statusMailerCapture{}
	svc := requestServiceForTest(repo, fakeAddressProvider{})
	svc.Mailer = mailer

	r, err := svc.Next(authContext(), "7", false)
	if err != nil {
		t.Fatal(err)
	}
	if mailer.email != "resident@example.com" || mailer.request.ID != r.ID || mailer.request.Status != "SENT" {
		t.Fatalf("status email was not sent to request owner: %+v", mailer)
	}
}

func authContext() context.Context {
	return WithCurrentUser(context.Background(), domain.User{ID: "99", Email: "resident@example.com", Roles: []string{"resident"}})
}

func requestServiceForTest(repo *captureRepository, addresses fakeAddressProvider) RequestService {
	houses := &fakeHouseRepository{byObjectID: map[string]domain.House{}, nextID: "42"}
	return RequestService{
		Repo: repo, Addresses: addresses, Houses: houses,
		Classifier: RuleClassifier{}, Router: RuleRouter{},
		Notifications: NotificationService{Client: integration.MockMaxClient{}}, Housing: integration.MockHousingSystemGateway{},
	}
}
