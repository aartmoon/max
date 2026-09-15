package domain

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("обращение не найдено")
var ErrConflict = errors.New("переход статуса недоступен")

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type House struct {
	ID      string `json:"id"`
	Address string `json:"address"`
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
	ResponsibleOrganization   string    `json:"responsibleOrganization"`
	Text                      string    `json:"text"`
	HasPhoto                  bool      `json:"hasPhoto"`
	Photo                     []byte    `json:"-"`
	PhotoType                 string    `json:"-"`
}
type RequestStatusHistory struct {
	ID        int64     `json:"id"`
	RequestID string    `json:"requestId"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}
