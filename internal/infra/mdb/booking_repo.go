package mdb

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type bookingRepo struct {
	collection *mongo.Collection
}

func NewBookingRepo(db *mongo.Database, config *config.BookingCollectionConfig) port.BookingRepo {
	return &bookingRepo{collection: db.Collection(config.Name)}
}

func (b *bookingRepo) FindByGuestId(ctx context.Context, guestId bson.ObjectID) ([]documents.Booking, error) {
	if err := validation.New().NotNilObjectID("guestId", guestId).Validate(); err != nil {
		return nil, err
	}

	cursor, err := b.collection.Find(ctx, bson.M{"main_guest": guestId})
	if err != nil {
		return nil, fmt.Errorf("%w: could not find bookings for guestId=%s: %v",
			dbErrors.ErrBookingRepo, guestId.Hex(), err)
	}

	//goland:noinspection GoUnhandledErrorResult
	defer cursor.Close(ctx)

	var bookings = make([]documents.Booking, 0)
	if err := cursor.All(ctx, &bookings); err != nil {
		return nil, fmt.Errorf("%w: could not find bookings for guestId=%s: %v",
			dbErrors.ErrBookingRepo, guestId.Hex(), err)
	}

	return bookings, nil
}
