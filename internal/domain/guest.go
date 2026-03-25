package domain

import (
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Guest struct {
	Id             bson.ObjectID
	documentId     string
	givenNames     string
	surname        string
	email          string
	BillingAddress string
	createdAt      *time.Time
	lastUpdate     *time.Time
}

func NewGuest(id bson.ObjectID, documentId string, givenNames string, surname string, email string, billingAddress string, createdTime *time.Time, lastUpdate *time.Time) (Guest, error) {
	if err := validation.New().
		NotNilObjectID("Id", id).
		NotBlank("documentId", documentId).
		NotBlank("givenNames", givenNames).
		NotBlank("surname", surname).
		EmailFormat(email).
		NotBlank("billingAddress", billingAddress).
		NotZeroValue("createdTime", createdTime).Validate(); err != nil {
		return Guest{}, err
	}

	return Guest{
		id,
		documentId,
		givenNames,
		surname,
		email,
		billingAddress,
		createdTime,
		lastUpdate}, nil
}

func (g *Guest) ToDto() dto.GuestDto {
	return dto.GuestDto{
		DocumentId:     g.documentId,
		GivenNames:     g.givenNames,
		Surname:        g.surname,
		Email:          g.email,
		BillingAddress: g.BillingAddress,
		CreatedAt:      g.createdAt,
		LastUpdate:     g.lastUpdate,
	}
}

func (g *Guest) ToMongoDoc() documents.Guest {
	return documents.Guest{
		Id:             g.Id,
		DocumentId:     g.documentId,
		GivenNames:     g.givenNames,
		Surname:        g.surname,
		Email:          g.email,
		BillingAddress: g.BillingAddress,
		CreatedAt:      g.createdAt,
		LastUpdate:     g.lastUpdate,
	}
}
