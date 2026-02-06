package domain

import (
	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Guest struct {
	Id         bson.ObjectID
	documentId string
	givenNames string
	surname    string
	email      string
}

func NewGuest(id bson.ObjectID, documentId string, givenNames string, surname string, email string) (Guest, error) {
	if err := validation.New().
		NotNilObjectID("Id", id).
		NotBlank("documentId", documentId).
		NotBlank("givenNames", givenNames).
		NotBlank("surname", surname).
		EmailFormat(email).Validate(); err != nil {
		return Guest{}, err
	}

	return Guest{id, documentId, givenNames, surname, email}, nil
}

func (g *Guest) ToDto() dto.GuestDto {
	return dto.GuestDto{
		DocumentId: g.documentId,
		GivenNames: g.givenNames,
		Surname:    g.surname,
		Email:      g.email,
	}
}

func (g *Guest) ToMongoDoc() documents.Guest {
	return documents.Guest{
		Id:         g.Id,
		DocumentId: g.documentId,
		GivenNames: g.givenNames,
		Surname:    g.surname,
		Email:      g.email,
	}
}
