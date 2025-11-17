package port

import (
	"context"
	"guestManager/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GuestRepo interface {
	GetById(ctx context.Context, id primitive.ObjectID) (domain.Guest, error)
	GetByDocument(ctx context.Context, documentId string) (domain.Guest, error)
	Add(ctx context.Context, newGuest domain.Guest) (primitive.ObjectID, error)
	Update(ctx context.Context, id primitive.ObjectID, guest domain.Guest) (domain.Guest, error)
}

type CottageRepo interface {
	UpdateCurrentGuest(ctx context.Context, roomName string, guestId primitive.ObjectID) error
}

type BookingRepo interface {
	FindByGuestId(ctx context.Context, guestId primitive.ObjectID) ([]domain.Booking, error)
}
