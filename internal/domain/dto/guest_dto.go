package dto

import "time"

type GuestDto struct {
	DocumentId string     `json:"document_id"`
	GivenNames string     `json:"given_names"`
	Surname    string     `json:"surname"`
	Email      string     `json:"email"`
	CreatedAt  *time.Time `json:"created_at"`
	LastUpdate *time.Time `json:"last_update,omitempty"`
}
