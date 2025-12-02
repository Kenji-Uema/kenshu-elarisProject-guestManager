package app

import (
	"context"
	"guestManager/internal/domain"
	"guestManager/internal/port"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GuestService interface {
	GetById(ctx context.Context, id primitive.ObjectID) (domain.Guest, error)
	GetByDocument(ctx context.Context, documentId string) (domain.Guest, error)
	Add(ctx context.Context, guest domain.Guest) (primitive.ObjectID, error)
	Update(ctx context.Context, id primitive.ObjectID, guest domain.Guest) (domain.Guest, error)
}

type guestService struct {
	repo port.GuestRepo
}

func NewGuestService(guestRepo port.GuestRepo) GuestService {
	return &guestService{repo: guestRepo}
}

func (g guestService) GetById(ctx context.Context, id primitive.ObjectID) (domain.Guest, error) {
	guestDoc, err := g.repo.GetById(ctx, id)

	if err != nil {
		return domain.Guest{}, err
	}

	guest, err := guestDoc.ToDomain()

	if err != nil {
		return domain.Guest{}, err
	}

	return guest, nil
}

func (g guestService) GetByDocument(ctx context.Context, documentId string) (domain.Guest, error) {
	return g.repo.GetByDocument(ctx, documentId)
}

func (g guestService) Add(ctx context.Context, guest domain.Guest) (primitive.ObjectID, error) {
	return g.repo.Add(ctx, guest)
}

func (g guestService) Update(ctx context.Context, id primitive.ObjectID, guest domain.Guest) (domain.Guest, error) {
	return g.repo.Update(ctx, id, guest)
}
