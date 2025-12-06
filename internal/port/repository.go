package port

import (
	"context"
	"guestManager/internal/domain/documents"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GuestRepo interface {
	GetById(ctx context.Context, id primitive.ObjectID) (documents.Guest, error)
	GetByDocument(ctx context.Context, documentId string) (documents.Guest, error)
	Add(ctx context.Context, newGuest documents.Guest) (primitive.ObjectID, error)
	Update(ctx context.Context, id primitive.ObjectID, guest documents.Guest) (documents.Guest, error)
}

type CottageRepo interface {
	UpdateCurrentGuest(ctx context.Context, roomName string, guestId primitive.ObjectID) error
}

type BookingRepo interface {
	FindByGuestId(ctx context.Context, guestId primitive.ObjectID) ([]documents.Booking, error)
}
