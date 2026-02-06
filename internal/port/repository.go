package port

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type GuestRepo interface {
	GetById(ctx context.Context, id bson.ObjectID) (documents.Guest, error)
	GetByDocument(ctx context.Context, documentId string) (documents.Guest, error)
	Add(ctx context.Context, newGuest documents.Guest) (bson.ObjectID, error)
	Update(ctx context.Context, id bson.ObjectID, guest documents.Guest) (documents.Guest, error)
}

type CottageRepo interface {
	UpdateCurrentGuest(ctx context.Context, roomName string, guestId bson.ObjectID) error
}

type BookingRepo interface {
	FindByGuestId(ctx context.Context, guestId bson.ObjectID) ([]documents.Booking, error)
}
