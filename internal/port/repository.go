package port

import (
	"context"
	"time"

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
	GetByName(ctx context.Context, roomName string) (documents.Cottage, error)
	UpdateCurrentGuest(ctx context.Context, roomName string, guestId bson.ObjectID) error
	ClearCurrentGuest(ctx context.Context, roomName string) error
}

type BookingRepo interface {
	FindByGuestId(ctx context.Context, guestId bson.ObjectID) ([]documents.Booking, error)
	FindByCheckInDate(ctx context.Context, date time.Time) ([]documents.Booking, error)
	FindByGuestIdAndCheckIn(ctx context.Context, guestId bson.ObjectID, checkIn time.Time) (documents.Booking, error)
	FindByGuestIdAndBookingNumber(ctx context.Context, guestId bson.ObjectID, bookingNumber string) (documents.Booking, error)
}
