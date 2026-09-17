package domain

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("объект не найден")
var ErrConflict = errors.New("переход статуса недоступен")
var ErrInvalidAddressID = errors.New("некорректный идентификатор адреса ГАР")
var ErrInvalidHouse = errors.New("выбранный объект не является домом")
var ErrInvalidApartment = errors.New("выбранная квартира не относится к указанному дому")

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type House struct {
	ID              string     `json:"id"`
	GARObjectID     string     `json:"garObjectId,omitempty"`
	ObjectGUID      string     `json:"objectGuid,omitempty"`
	Address         string     `json:"address"`
	CadastralNumber *string    `json:"cadastralNumber"`
	CreatedAt       *time.Time `json:"createdAt,omitempty"`
	UpdatedAt       *time.Time `json:"updatedAt,omitempty"`
	TotalArea       float64    `json:"totalArea"`
	LivingArea      float64    `json:"livingArea"`
	Floors          int        `json:"floors"`
	Entrances       int        `json:"entrances"`
	Apartments      int        `json:"apartments"`
	YearBuilt       int        `json:"yearBuilt"`
	Organization    string     `json:"organization"`
	Manager         string     `json:"manager"`
	Contact         string     `json:"contact"`
}
type AddressSearch struct {
	Query          string
	Kind           string
	ParentObjectID *int64
	Limit          int
}
type AddressSuggestion struct {
	ObjectID       string `json:"objectId"`
	ObjectGUID     string `json:"objectGuid,omitempty"`
	ParentObjectID string `json:"parentObjectId,omitempty"`
	ObjectKind     string `json:"objectKind"`
	DisplayName    string `json:"displayName"`
	FullAddress    string `json:"fullAddress"`
}
type AddressInfo struct {
	ObjectID       string `json:"objectId"`
	ObjectGUID     string `json:"objectGuid,omitempty"`
	ParentObjectID string `json:"parentObjectId,omitempty"`
	ObjectKind     string `json:"objectKind"`
	DisplayName    string `json:"displayName"`
	FullAddress    string `json:"fullAddress"`
	IsActive       bool   `json:"isActive"`
}
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Request struct {
	ID                        string    `json:"id"`
	UserID                    string    `json:"userId"`
	HouseID                   string    `json:"houseId"`
	Description               string    `json:"description"`
	ProblemType               string    `json:"problemType"`
	ResponsibleOrganizationID string    `json:"responsibleOrganizationId"`
	Status                    string    `json:"status"`
	Deadline                  time.Time `json:"deadline"`
	CreatedAt                 time.Time `json:"createdAt"`
	Kind                      string    `json:"kind"`
	Address                   string    `json:"address"`
	HouseObjectID             string    `json:"houseObjectId"`
	HouseObjectGUID           string    `json:"houseObjectGuid,omitempty"`
	ApartmentObjectID         string    `json:"apartmentObjectId,omitempty"`
	ApartmentObjectGUID       string    `json:"apartmentObjectGuid,omitempty"`
	ResponsibleOrganization   string    `json:"responsibleOrganization"`
	Text                      string    `json:"text"`
	HasPhoto                  bool      `json:"hasPhoto"`
	Photo                     []byte    `json:"-"`
	PhotoType                 string    `json:"-"`
}
type RequestStatusHistory struct {
	Comment   string    `json:"comment"`
	Actor     string    `json:"actor"`
	ID        int64     `json:"id"`
	RequestID string    `json:"requestId"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}
