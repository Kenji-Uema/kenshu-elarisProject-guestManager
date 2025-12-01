package mdb

import (
	"context"
	"fmt"
	"guestManager/internal/config"
	"guestManager/internal/domain"
	"guestManager/internal/domain/errors/dbErrors"
	"guestManager/internal/port"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type bookingRepo struct {
	collection *mongo.Collection
}

func NewBookingRepo(db *mongo.Database, config *config.BookingCollectionConfig) port.BookingRepo {
	return &bookingRepo{collection: db.Collection(config.Name)}
}

func (b *bookingRepo) FindByGuestId(ctx context.Context, guestId primitive.ObjectID) ([]domain.Booking, error) {
	cursor, err := b.collection.Find(ctx, bson.M{"main_guest": guestId})
	if err != nil {
		return nil, fmt.Errorf("%w: could not find bookings for guestId=%s: %v",
			dbErrors.ErrBookingRepo, guestId.Hex(), err)
	}

	//goland:noinspection GoUnhandledErrorResult
	defer cursor.Close(ctx)

	var bookings = make([]domain.Booking, 0)
	if err := cursor.All(ctx, &bookings); err != nil {
		return nil, fmt.Errorf("%w: could not find bookings for guestId=%s: %v",
			dbErrors.ErrBookingRepo, guestId.Hex(), err)
	}

	return bookings, nil
}
