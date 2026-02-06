package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type GuestService interface {
	GetById(ctx context.Context, id bson.ObjectID) (domain.Guest, error)
	GetByDocument(ctx context.Context, documentId string) (domain.Guest, error)
	Add(ctx context.Context, guest domain.Guest) (bson.ObjectID, error)
	Update(ctx context.Context, id bson.ObjectID, guest domain.Guest) (domain.Guest, error)
}

type guestService struct {
	repo port.GuestRepo
}

func NewGuestService(guestRepo port.GuestRepo) GuestService {
	return &guestService{repo: guestRepo}
}

func (g *guestService) GetById(ctx context.Context, id bson.ObjectID) (domain.Guest, error) {
	guestDoc, err := g.repo.GetById(ctx, id)
	if err != nil {
		return domain.Guest{}, err
	}

	guest, err := domain.NewGuest(guestDoc.Id, guestDoc.DocumentId, guestDoc.GivenNames, guestDoc.Surname, guestDoc.Email)
	if err != nil {
		var validationErr *validationErrors.ErrValidationConstrain
		if errors.As(err, &validationErr) {
			return domain.Guest{}, err
		}

		return domain.Guest{}, fmt.Errorf("guestService.GetById(%s): mapping to domain: %w", id.Hex(), err)
	}

	return guest, nil
}

func (g *guestService) GetByDocument(ctx context.Context, documentId string) (domain.Guest, error) {
	guestDoc, err := g.repo.GetByDocument(ctx, documentId)
	if err != nil {
		return domain.Guest{}, err
	}

	guest, err := domain.NewGuest(guestDoc.Id, guestDoc.DocumentId, guestDoc.GivenNames, guestDoc.Surname, guestDoc.Email)
	if err != nil {
		var validationErr *validationErrors.ErrValidationConstrain
		if errors.As(err, &validationErr) {
			return domain.Guest{}, err
		}

		return domain.Guest{}, fmt.Errorf("guestService.GetByDocument(%s): mapping to domain: %w", documentId, err)
	}

	return guest, nil

}

func (g *guestService) Add(ctx context.Context, guest domain.Guest) (bson.ObjectID, error) {
	return g.repo.Add(ctx, guest.ToMongoDoc())
}

func (g *guestService) Update(ctx context.Context, id bson.ObjectID, guest domain.Guest) (domain.Guest, error) {
	updatedGuestDoc, err := g.repo.Update(ctx, id, guest.ToMongoDoc())
	if err != nil {
		return domain.Guest{}, err
	}

	updatedGuest, err := domain.NewGuest(updatedGuestDoc.Id, updatedGuestDoc.DocumentId, updatedGuestDoc.GivenNames,
		updatedGuestDoc.Surname, updatedGuestDoc.Email)
	if err != nil {
		var validationErr *validationErrors.ErrValidationConstrain
		if errors.As(err, &validationErr) {
			return domain.Guest{}, err
		}

		return domain.Guest{}, fmt.Errorf("guestService.Update(%s): mapping to domain: %w", id.Hex(), err)
	}

	return updatedGuest, nil
}
