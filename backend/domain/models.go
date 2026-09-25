package domain

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrNotFound = errors.New("объект не найден")
var ErrConflict = errors.New("переход статуса недоступен")
var ErrInvalidAddressID = errors.New("некорректный идентификатор адреса ГАР")
var ErrInvalidHouse = errors.New("выбранный объект не является домом")
var ErrInvalidApartment = errors.New("выбранная квартира не относится к указанному дому")
var ErrHouseProfileNotFound = errors.New("сведения о доме не найдены в ГИС ЖКХ")
var ErrHouseProfileUnavailable = errors.New("ГИС ЖКХ временно недоступна")
var ErrUnauthorized = errors.New("требуется вход")
var ErrForbidden = errors.New("недостаточно прав")

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

type User struct {
	ID             string   `json:"id"`
	Email          string   `json:"email,omitempty"`
	Name           string   `json:"name"`
	Roles          []string `json:"roles,omitempty"`
	OrganizationID *string  `json:"organizationId,omitempty"`
}

type AuthSession struct {
	Token string `json:"-"`
	User  User   `json:"user"`
}

type HouseCharacteristics struct {
	HouseTypeCode                       *string    `json:"houseTypeCode"`
	HouseType                           *string    `json:"houseType"`
	Status                              *string    `json:"status"`
	ProjectSeries                       *string    `json:"projectSeries"`
	Condition                           *string    `json:"condition"`
	LifecycleStage                      *string    `json:"lifecycleStage"`
	YearBuilt                           *int       `json:"yearBuilt"`
	OperationYear                       *int       `json:"operationYear"`
	ReconstructionYear                  *int       `json:"reconstructionYear"`
	DeteriorationPercent                *float64   `json:"deteriorationPercent"`
	DeteriorationDate                   *time.Time `json:"deteriorationDate"`
	WallMaterial                        *string    `json:"wallMaterial"`
	EnergyEfficiency                    *string    `json:"energyEfficiency"`
	TotalArea                           *float64   `json:"totalArea"`
	LivingArea                          *float64   `json:"livingArea"`
	NonResidentialArea                  *float64   `json:"nonResidentialArea"`
	ResidentialPremises                 *int       `json:"residentialPremises"`
	ResidentialPremisesArea             *float64   `json:"residentialPremisesArea"`
	ResidentialPremisesWithRealty       *int       `json:"residentialPremisesWithRealty"`
	ResidentialPremisesWithRealtyArea   *float64   `json:"residentialPremisesWithRealtyArea"`
	NonResidentialPremises              *int       `json:"nonResidentialPremises"`
	NonResidentialPremisesArea          *float64   `json:"nonResidentialPremisesArea"`
	NonResidentialPremisesNotCommon     *int       `json:"nonResidentialPremisesNotCommon"`
	NonResidentialPremisesNotCommonArea *float64   `json:"nonResidentialPremisesNotCommonArea"`
	Floors                              *int       `json:"floors"`
	Entrances                           *int       `json:"entrances"`
	OwnersOrShares                      *int       `json:"ownersOrShares"`
}

type HouseManagement struct {
	Method                   *string    `json:"method"`
	OrganizationGUID         *string    `json:"organizationGuid"`
	ShortName                *string    `json:"shortName"`
	FullName                 *string    `json:"fullName"`
	Address                  *string    `json:"address"`
	Phone                    *string    `json:"phone"`
	Website                  *string    `json:"website"`
	OrganizationType         *string    `json:"organizationType"`
	RegistryOrganizationGUID *string    `json:"registryOrganizationGuid"`
	INN                      *string    `json:"inn"`
	OGRN                     *string    `json:"ogrn"`
	Chief                    *string    `json:"chief"`
	ContractStart            *time.Time `json:"contractStart"`
	ContractEnd              *time.Time `json:"contractEnd"`
}

type HouseDataSource struct {
	Name      string    `json:"name"`
	Available bool      `json:"available"`
	UpdatedAt time.Time `json:"updatedAt"`
	Stale     bool      `json:"stale"`
}

type House struct {
	ID              string               `json:"id"`
	GARObjectID     string               `json:"garObjectId,omitempty"`
	ObjectGUID      string               `json:"objectGuid,omitempty"`
	Address         string               `json:"address"`
	CadastralNumber *string              `json:"cadastralNumber"`
	CreatedAt       *time.Time           `json:"createdAt,omitempty"`
	UpdatedAt       *time.Time           `json:"updatedAt,omitempty"`
	TotalArea       *float64             `json:"totalArea"`
	LivingArea      *float64             `json:"livingArea"`
	Floors          *int                 `json:"floors"`
	Entrances       *int                 `json:"entrances"`
	Apartments      *int                 `json:"apartments"`
	YearBuilt       *int                 `json:"yearBuilt"`
	Organization    *string              `json:"organization"`
	Manager         *string              `json:"manager"`
	Contact         *string              `json:"contact"`
	DataSource      string               `json:"dataSource,omitempty"`
	DataUpdatedAt   *time.Time           `json:"dataUpdatedAt,omitempty"`
	Stale           bool                 `json:"stale"`
	Characteristics HouseCharacteristics `json:"characteristics"`
	Management      HouseManagement      `json:"management"`
	DataSources     []HouseDataSource    `json:"dataSources"`
}

type HouseProfile struct {
	HouseID                string               `json:"houseId,omitempty"`
	GISHouseGUID           string               `json:"gisHouseGuid"`
	GISHouseType           string               `json:"gisHouseType"`
	CadastralNumber        *string              `json:"cadastralNumber"`
	TotalArea              *float64             `json:"totalArea"`
	LivingArea             *float64             `json:"livingArea"`
	Floors                 *int                 `json:"floors"`
	Entrances              *int                 `json:"entrances"`
	Apartments             *int                 `json:"apartments"`
	YearBuilt              *int                 `json:"yearBuilt"`
	Organization           *string              `json:"organization"`
	Manager                *string              `json:"manager"`
	Contact                *string              `json:"contact"`
	RawPayload             json.RawMessage      `json:"-"`
	SquarePayload          json.RawMessage      `json:"-"`
	SquareSummaryAvailable bool                 `json:"squareSummaryAvailable"`
	FetchedAt              time.Time            `json:"fetchedAt"`
	Stale                  bool                 `json:"stale"`
	Characteristics        HouseCharacteristics `json:"characteristics"`
	Management             HouseManagement      `json:"management"`
	DataSources            []HouseDataSource    `json:"dataSources"`
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
type UserApartment struct {
	ID                  string `json:"id"`
	UserID              string `json:"userId"`
	HouseObjectID       string `json:"houseObjectId"`
	HouseObjectGUID     string `json:"houseObjectGuid,omitempty"`
	ApartmentObjectID   string `json:"apartmentObjectId,omitempty"`
	ApartmentObjectGUID string `json:"apartmentObjectGuid,omitempty"`
	Address             string `json:"address"`
	Label               string `json:"label"`
	IsDefault           bool   `json:"isDefault"`
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
