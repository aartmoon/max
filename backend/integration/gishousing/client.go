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

type houseDTO struct {
	GUID          string       `json:"guid"`
	HouseHMGUID   string       `json:"houseHMGuid"`
	Cadastre      *string      `json:"cadastreNumber"`
	Total         *float64     `json:"totalSquare"`
	Residential   *float64     `json:"residentialSquare"`
	Floors        *flexibleInt `json:"floorCountMax"`
	Entrances     *flexibleInt `json:"entranceCount"`
	Apartments    *flexibleInt `json:"residentialPremiseCount"`
	BuildingYear  *flexibleInt `json:"buildingYear"`
	OperationYear *flexibleInt `json:"operationYear"`
	HouseType     struct {
		Code string `json:"code"`
	} `json:"houseType"`
	Address struct {
		House struct {
			GUID string `json:"houseGuid"`
		} `json:"house"`
	} `json:"address"`
	Management struct {
		ShortName string `json:"shortName"`
		FullName  string `json:"fullName"`
		Phone     string `json:"phone"`
	} `json:"managementOrganization"`
	ChiefLastName   string `json:"chiefLastName"`
	ChiefFirstName  string `json:"chiefFirstName"`
	ChiefMiddleName string `json:"chiefMiddleName"`
}

type flexibleInt int

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

func intPointer(v *flexibleInt) *int {
	if v == nil {
		return nil
	}
	value := int(*v)
	return &value
}

func (c Client) FetchHouse(ctx context.Context, fiasGUID string) (domain.HouseProfile, error) {
	lookupURL := c.BaseURL + "/homemanagement/api/rest/services/houses/public/houses/searchByFiasHouseCodeList/" + url.PathEscape(fiasGUID) + "?useReadOnlyDataSource=true"
	var lookup struct {
		HouseList []houseDTO `json:"houseList"`
	}
	if _, err := c.getJSON(ctx, lookupURL, &lookup); err != nil {
		return domain.HouseProfile{}, err
	}
	var match *houseDTO
	for i := range lookup.HouseList {
		if strings.EqualFold(lookup.HouseList[i].Address.House.GUID, fiasGUID) {
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
	var detail houseDTO
	raw, err := c.getJSON(ctx, detailURL, &detail)
	if err != nil {
		return domain.HouseProfile{}, err
	}
	if detail.GUID == "" {
		detail.GUID = guid
	}
	if detail.HouseType.Code == "" {
		detail.HouseType.Code = match.HouseType.Code
	}
	mergeMissing(&detail, *match)
	year := detail.BuildingYear
	if year == nil {
		year = detail.OperationYear
	}
	organization := optional(first(detail.Management.ShortName, detail.Management.FullName))
	manager := optional(strings.TrimSpace(strings.Join([]string{detail.ChiefLastName, detail.ChiefFirstName, detail.ChiefMiddleName}, " ")))
	return domain.HouseProfile{
		GISHouseGUID: detail.GUID, GISHouseType: detail.HouseType.Code,
		CadastralNumber: detail.Cadastre, TotalArea: detail.Total, LivingArea: detail.Residential,
		Floors: intPointer(detail.Floors), Entrances: intPointer(detail.Entrances), Apartments: intPointer(detail.Apartments),
		YearBuilt: intPointer(year), Organization: organization, Manager: manager, Contact: optional(detail.Management.Phone),
		RawPayload: raw, FetchedAt: c.now().UTC(),
	}, nil
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

func mergeMissing(dst *houseDTO, src houseDTO) {
	if dst.Cadastre == nil {
		dst.Cadastre = src.Cadastre
	}
	if dst.Total == nil {
		dst.Total = src.Total
	}
	if dst.Residential == nil {
		dst.Residential = src.Residential
	}
	if dst.Floors == nil {
		dst.Floors = src.Floors
	}
	if dst.Entrances == nil {
		dst.Entrances = src.Entrances
	}
	if dst.Apartments == nil {
		dst.Apartments = src.Apartments
	}
	if dst.BuildingYear == nil {
		dst.BuildingYear = src.BuildingYear
	}
	if dst.OperationYear == nil {
		dst.OperationYear = src.OperationYear
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
