package gishousing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tvoydom/domain"
)

const maxResponseBytes = 2 << 20

type Client struct {
	BaseURL string
	HTTP    *http.Client
	Now     func() time.Time
}

func NewClient(baseURL string, timeout time.Duration, httpClient *http.Client) Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	copy := *httpClient
	copy.Timeout = timeout
	return Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: &copy, Now: time.Now}
}

type houseTypeDTO struct {
	Code string `json:"code"`
	Name string `json:"houseTypeName"`
}

type managementDTO struct {
	GUID                     string `json:"guid"`
	ShortName                string `json:"shortName"`
	FullName                 string `json:"fullName"`
	Address                  string `json:"orgAddress"`
	Phone                    string `json:"phone"`
	Website                  string `json:"url"`
	OrganizationType         string `json:"organizationType"`
	RegistryOrganizationGUID string `json:"registryOrganizationRootEntityGuid"`
	INN                      string `json:"inn"`
	OGRN                     string `json:"ogrn"`
}

type houseCommonDTO struct {
	Cadastre                     *string        `json:"cadastreNumber"`
	Total                        *flexibleFloat `json:"totalSquare"`
	Residential                  *flexibleFloat `json:"residentialSquare"`
	FloorCount                   *flexibleInt   `json:"floorCount"`
	FloorCountMax                *flexibleInt   `json:"floorCountMax"`
	Entrances                    *flexibleInt   `json:"entranceCount"`
	ResidentialPremises          *flexibleInt   `json:"residentialPremiseCount"`
	ResidentialPremisesActual    *flexibleInt   `json:"residentialPremiseActualCount"`
	ResidentialPremisesConfirmed *flexibleInt   `json:"residentialPremiseConfirmedCount"`
	BuildingYear                 *flexibleInt   `json:"buildingYear"`
	OperationYear                *flexibleInt   `json:"operationYear"`
	HouseType                    houseTypeDTO   `json:"houseType"`
	Management                   managementDTO  `json:"managementOrganization"`
	ChiefLastName                string         `json:"chiefLastName"`
	ChiefFirstName               string         `json:"chiefFirstName"`
	ChiefMiddleName              string         `json:"chiefMiddleName"`
}

type houseLookupDTO struct {
	houseCommonDTO
	GUID        string `json:"guid"`
	HouseHMGUID string `json:"houseHMGuid"`
	Address     struct {
		House struct {
			GUID string `json:"houseGuid"`
		} `json:"house"`
	} `json:"address"`
	House struct {
		Code string `json:"code"`
	} `json:"house"`
}

type houseDetailDTO struct {
	houseCommonDTO
	GUID               string         `json:"guid"`
	Status             string         `json:"status"`
	ReconstructionYear *flexibleInt   `json:"reconstructionYear"`
	PlanSeries         string         `json:"planSeries"`
	Deterioration      *flexibleFloat `json:"deterioration"`
	DeteriorationDate  *flexibleDate  `json:"deteriorationDate"`
	WallMaterial       string         `json:"intWallMaterialList"`
	EnergyEfficiency   struct {
		Code             string `json:"code"`
		Value            string `json:"value"`
		Name             string `json:"name"`
		EnergyEfficiency string `json:"energyEfficiency"`
	} `json:"houseEnergyEfficiency"`
	HouseCondition struct {
		Name string `json:"houseCondition"`
	} `json:"houseCondition"`
	LifecycleStage struct {
		Name string `json:"lifeCycleStage"`
	} `json:"lifeCycleStage"`
	ManagementType struct {
		Name string `json:"houseManagementTypeName"`
	} `json:"houseManagementType"`
	ManagementContractDate *flexibleDate `json:"managementContractDate"`
	EndContractDate        *flexibleDate `json:"endContractDate"`
}

type squareSummaryDTO struct {
	Total                               *flexibleFloat `json:"totalSquare"`
	Residential                         *flexibleFloat `json:"residentialSquare"`
	NonResidential                      *flexibleFloat `json:"nonresidentialSquare"`
	ResidentialPremises                 *flexibleInt   `json:"residentialPremiseCount"`
	ResidentialPremisesArea             *flexibleFloat `json:"residentialPremiseSquare"`
	ResidentialPremisesWithRealty       *flexibleInt   `json:"residentialPremiseWithRelatedRealtyCount"`
	ResidentialPremisesWithRealtyArea   *flexibleFloat `json:"residentialPremiseWithRelatedRealtySquare"`
	NonResidentialPremises              *flexibleInt   `json:"nonresidentialPremiseCount"`
	NonResidentialPremisesArea          *flexibleFloat `json:"nonresidentialPremiseSquare"`
	NonResidentialPremisesNotCommon     *flexibleInt   `json:"nonresidentialPremiseNotCommonCount"`
	NonResidentialPremisesNotCommonArea *flexibleFloat `json:"nonresidentialPremiseNotCommonSquare"`
	OwnersOrShares                      *flexibleInt   `json:"shareOwnersNumber"`
}

type flexibleInt int

type flexibleFloat float64

type flexibleDate time.Time

func (v *flexibleFloat) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "null" {
		return nil
	}
	if strings.HasPrefix(raw, `"`) {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		raw = strings.TrimSpace(text)
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fmt.Errorf("expected number, got %s", string(data))
	}
	*v = flexibleFloat(value)
	return nil
}

func (v *flexibleInt) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "null" {
		return nil
	}
	if strings.HasPrefix(raw, `"`) {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		raw = strings.TrimSpace(text)
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("expected integer, got %s", string(data))
	}
	*v = flexibleInt(value)
	return nil
}

func (v *flexibleDate) UnmarshalJSON(data []byte) error {
	var text string
	if string(data) == "null" {
		return nil
	}
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("expected date string, got %s", string(data))
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	for _, layout := range []string{"02.01.2006", "2006-01-02", time.RFC3339} {
		if parsed, err := time.Parse(layout, text); err == nil {
			*v = flexibleDate(parsed)
			return nil
		}
	}
	return fmt.Errorf("expected supported date, got %q", text)
}

func intPointer(v *flexibleInt) *int {
	if v == nil {
		return nil
	}
	value := int(*v)
	return &value
}

func floatPointer(v *flexibleFloat) *float64 {
	if v == nil {
		return nil
	}
	value := float64(*v)
	return &value
}

func datePointer(v *flexibleDate) *time.Time {
	if v == nil || time.Time(*v).IsZero() {
		return nil
	}
	value := time.Time(*v)
	return &value
}

func (c Client) FetchHouse(ctx context.Context, fiasGUID string) (domain.HouseProfile, error) {
	lookupURL := c.BaseURL + "/homemanagement/api/rest/services/houses/public/houses/searchByFiasHouseCodeList/" + url.PathEscape(fiasGUID) + "?useReadOnlyDataSource=true"
	var lookup struct {
		HouseList []houseLookupDTO `json:"houseList"`
	}
	if _, err := c.getJSON(ctx, lookupURL, &lookup); err != nil {
		return domain.HouseProfile{}, err
	}
	var match *houseLookupDTO
	for i := range lookup.HouseList {
		if strings.EqualFold(fiasHouseGUID(lookup.HouseList[i]), fiasGUID) {
			match = &lookup.HouseList[i]
			break
		}
	}
	if match == nil {
		return domain.HouseProfile{}, domain.ErrHouseProfileNotFound
	}
	guid := match.GUID
	if guid == "" {
		guid = match.HouseHMGUID
	}
	if guid == "" || match.HouseType.Code == "" {
		return domain.HouseProfile{}, fmt.Errorf("%w: incomplete lookup response", domain.ErrHouseProfileUnavailable)
	}
	detailURL := c.BaseURL + "/homemanagement/api/rest/services/houses/public/" + url.PathEscape(match.HouseType.Code) + "/" + url.PathEscape(guid)
	var detail houseDetailDTO
	rawDetail, err := c.getJSON(ctx, detailURL, &detail)
	if err != nil {
		return domain.HouseProfile{}, err
	}
	if detail.GUID == "" {
		detail.GUID = guid
	}
	if detail.HouseType.Code == "" {
		detail.HouseType.Code = match.HouseType.Code
	}
	mergeCommonMissing(&detail.houseCommonDTO, match.houseCommonDTO)
	profile := profileFromDetail(detail, rawDetail, c.now().UTC())
	squareURL := c.BaseURL + "/homemanagement/api/rest/services/houses/public/get-house-square-data/" + url.PathEscape(profile.GISHouseGUID)
	var square squareSummaryDTO
	if rawSquare, squareErr := c.getJSON(ctx, squareURL, &square); squareErr == nil {
		mergeSquareSummary(&profile, square)
		profile.SquarePayload = rawSquare
		profile.SquareSummaryAvailable = true
	}
	syncLegacyFields(&profile)
	profile.DataSources = []domain.HouseDataSource{
		{Name: "ГИС ЖКХ · карточка дома", Available: true, UpdatedAt: profile.FetchedAt},
		{Name: "ГИС ЖКХ · сводка помещений", Available: profile.SquareSummaryAvailable, UpdatedAt: profile.FetchedAt},
	}
	return profile, nil
}

func fiasHouseGUID(house houseLookupDTO) string {
	if house.Address.House.GUID != "" {
		return house.Address.House.GUID
	}
	return house.House.Code
}

func (c Client) getJSON(ctx context.Context, endpoint string, target any) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrHouseProfileUnavailable, err)
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", "TvoyDom/1.0 (+public GIS housing profile)")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrHouseProfileUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: HTTP %d", domain.ErrHouseProfileUnavailable, resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil || len(body) > maxResponseBytes {
		return nil, fmt.Errorf("%w: response is unreadable or too large", domain.ErrHouseProfileUnavailable)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON: %v", domain.ErrHouseProfileUnavailable, err)
	}
	return json.RawMessage(body), nil
}

func (c Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}
func optional(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
func first(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func profileFromDetail(detail houseDetailDTO, raw json.RawMessage, fetchedAt time.Time) domain.HouseProfile {
	year := firstInt(detail.BuildingYear, detail.OperationYear)
	characteristics := domain.HouseCharacteristics{
		HouseTypeCode:        optional(detail.HouseType.Code),
		HouseType:            optional(detail.HouseType.Name),
		Status:               optional(detail.Status),
		ProjectSeries:        optional(detail.PlanSeries),
		Condition:            optional(detail.HouseCondition.Name),
		LifecycleStage:       optional(detail.LifecycleStage.Name),
		YearBuilt:            intPointer(year),
		OperationYear:        intPointer(detail.OperationYear),
		ReconstructionYear:   intPointer(detail.ReconstructionYear),
		DeteriorationPercent: floatPointer(detail.Deterioration),
		DeteriorationDate:    datePointer(detail.DeteriorationDate),
		WallMaterial:         optional(detail.WallMaterial),
		EnergyEfficiency:     optional(first(detail.EnergyEfficiency.EnergyEfficiency, detail.EnergyEfficiency.Name, detail.EnergyEfficiency.Value, detail.EnergyEfficiency.Code)),
		TotalArea:            floatPointer(detail.Total),
		LivingArea:           floatPointer(detail.Residential),
		ResidentialPremises:  intPointer(firstInt(detail.ResidentialPremisesActual, detail.ResidentialPremisesConfirmed, detail.ResidentialPremises)),
		Floors:               intPointer(firstInt(detail.FloorCount, detail.FloorCountMax)),
		Entrances:            intPointer(detail.Entrances),
	}
	management := domain.HouseManagement{
		Method:                   optional(detail.ManagementType.Name),
		OrganizationGUID:         optional(detail.Management.GUID),
		ShortName:                optional(detail.Management.ShortName),
		FullName:                 optional(detail.Management.FullName),
		Address:                  optional(detail.Management.Address),
		Phone:                    optional(detail.Management.Phone),
		Website:                  optional(detail.Management.Website),
		OrganizationType:         optional(detail.Management.OrganizationType),
		RegistryOrganizationGUID: optional(detail.Management.RegistryOrganizationGUID),
		INN:                      optional(detail.Management.INN),
		OGRN:                     optional(detail.Management.OGRN),
		Chief:                    optional(strings.TrimSpace(strings.Join([]string{detail.ChiefLastName, detail.ChiefFirstName, detail.ChiefMiddleName}, " "))),
		ContractStart:            datePointer(detail.ManagementContractDate),
		ContractEnd:              datePointer(detail.EndContractDate),
	}
	return domain.HouseProfile{
		GISHouseGUID:    detail.GUID,
		GISHouseType:    detail.HouseType.Code,
		CadastralNumber: detail.Cadastre,
		RawPayload:      raw,
		FetchedAt:       fetchedAt,
		Characteristics: characteristics,
		Management:      management,
	}
}

func mergeSquareSummary(profile *domain.HouseProfile, square squareSummaryDTO) {
	characteristics := &profile.Characteristics
	assignFloat(&characteristics.TotalArea, square.Total)
	assignFloat(&characteristics.LivingArea, square.Residential)
	assignFloat(&characteristics.NonResidentialArea, square.NonResidential)
	assignInt(&characteristics.ResidentialPremises, square.ResidentialPremises)
	assignFloat(&characteristics.ResidentialPremisesArea, square.ResidentialPremisesArea)
	assignInt(&characteristics.ResidentialPremisesWithRealty, square.ResidentialPremisesWithRealty)
	assignFloat(&characteristics.ResidentialPremisesWithRealtyArea, square.ResidentialPremisesWithRealtyArea)
	assignInt(&characteristics.NonResidentialPremises, square.NonResidentialPremises)
	assignFloat(&characteristics.NonResidentialPremisesArea, square.NonResidentialPremisesArea)
	assignInt(&characteristics.NonResidentialPremisesNotCommon, square.NonResidentialPremisesNotCommon)
	assignFloat(&characteristics.NonResidentialPremisesNotCommonArea, square.NonResidentialPremisesNotCommonArea)
	assignInt(&characteristics.OwnersOrShares, square.OwnersOrShares)
}

func syncLegacyFields(profile *domain.HouseProfile) {
	profile.TotalArea = profile.Characteristics.TotalArea
	profile.LivingArea = profile.Characteristics.LivingArea
	profile.Floors = profile.Characteristics.Floors
	profile.Entrances = profile.Characteristics.Entrances
	profile.Apartments = profile.Characteristics.ResidentialPremises
	profile.YearBuilt = profile.Characteristics.YearBuilt
	profile.Organization = firstPointer(profile.Management.ShortName, profile.Management.FullName)
	profile.Manager = profile.Management.Chief
	profile.Contact = profile.Management.Phone
}

func firstInt(values ...*flexibleInt) *flexibleInt {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstPointer(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			return value
		}
	}
	return nil
}

func assignInt(target **int, value *flexibleInt) {
	if value != nil {
		*target = intPointer(value)
	}
}

func assignFloat(target **float64, value *flexibleFloat) {
	if value != nil {
		*target = floatPointer(value)
	}
}

func mergeCommonMissing(dst *houseCommonDTO, src houseCommonDTO) {
	if dst.Cadastre == nil {
		dst.Cadastre = src.Cadastre
	}
	if dst.Total == nil {
		dst.Total = src.Total
	}
	if dst.Residential == nil {
		dst.Residential = src.Residential
	}
	if dst.FloorCount == nil {
		dst.FloorCount = src.FloorCount
	}
	if dst.FloorCountMax == nil {
		dst.FloorCountMax = src.FloorCountMax
	}
	if dst.Entrances == nil {
		dst.Entrances = src.Entrances
	}
	if dst.ResidentialPremises == nil {
		dst.ResidentialPremises = src.ResidentialPremises
	}
	if dst.ResidentialPremisesActual == nil {
		dst.ResidentialPremisesActual = src.ResidentialPremisesActual
	}
	if dst.ResidentialPremisesConfirmed == nil {
		dst.ResidentialPremisesConfirmed = src.ResidentialPremisesConfirmed
	}
	if dst.BuildingYear == nil {
		dst.BuildingYear = src.BuildingYear
	}
	if dst.OperationYear == nil {
		dst.OperationYear = src.OperationYear
	}
	if dst.HouseType.Code == "" {
		dst.HouseType.Code = src.HouseType.Code
	}
	if dst.HouseType.Name == "" {
		dst.HouseType.Name = src.HouseType.Name
	}
	if dst.Management.ShortName == "" {
		dst.Management.ShortName = src.Management.ShortName
	}
	if dst.Management.FullName == "" {
		dst.Management.FullName = src.Management.FullName
	}
	if dst.Management.Phone == "" {
		dst.Management.Phone = src.Management.Phone
	}
	if dst.Management.GUID == "" {
		dst.Management.GUID = src.Management.GUID
	}
	if dst.Management.Address == "" {
		dst.Management.Address = src.Management.Address
	}
	if dst.Management.Website == "" {
		dst.Management.Website = src.Management.Website
	}
	if dst.Management.OrganizationType == "" {
		dst.Management.OrganizationType = src.Management.OrganizationType
	}
	if dst.Management.RegistryOrganizationGUID == "" {
		dst.Management.RegistryOrganizationGUID = src.Management.RegistryOrganizationGUID
	}
	if dst.Management.INN == "" {
		dst.Management.INN = src.Management.INN
	}
	if dst.Management.OGRN == "" {
		dst.Management.OGRN = src.Management.OGRN
	}
	if dst.ChiefLastName == "" {
		dst.ChiefLastName = src.ChiefLastName
	}
	if dst.ChiefFirstName == "" {
		dst.ChiefFirstName = src.ChiefFirstName
	}
	if dst.ChiefMiddleName == "" {
		dst.ChiefMiddleName = src.ChiefMiddleName
	}
}
