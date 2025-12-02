package domain

import (
	"guestManager/internal/domain/dto"
	"guestManager/internal/domain/errors/appErrors"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Guest struct {
	id         primitive.ObjectID
	documentId string
	givenNames string
	surname    string
	email      string
}

var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[A-Za-z]{2,}$`)

func NewGuest(id primitive.ObjectID, documentId string, givenNames string, surname string, email string) (Guest, error) {
	if id == primitive.NilObjectID {
		return Guest{}, &appErrors.ErrValidationConstrain{
			Field: "id", Message: "is required"}
	}

	if strings.TrimSpace(documentId) == "" {
		return Guest{}, &appErrors.ErrValidationConstrain{
			Field: "documentId", Message: "must not be empty"}
	}

	if strings.TrimSpace(givenNames) == "" {
		return Guest{}, &appErrors.ErrValidationConstrain{
			Field: "givenNames", Message: "must not be empty"}
	}

	if strings.TrimSpace(surname) == "" {
		return Guest{}, &appErrors.ErrValidationConstrain{
			Field: "surname", Message: "must not be empty"}
	}

	if !emailRe.MatchString(email) {
		return Guest{}, &appErrors.ErrValidationConstrain{
			Field:   "email",
			Message: "email not valid",
		}
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
