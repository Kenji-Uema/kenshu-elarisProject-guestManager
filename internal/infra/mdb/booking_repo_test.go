package mdb

import (
	"context"
	"errors"
	"guestManager/internal/domain/errors/validationErrors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func Test_bookingRepo_FindByGuestId_ReturnsBookings(t *testing.T) {
	setupAndRun("bookingRepo_FindByGuestId returns bookings", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		guestID, err := primitive.ObjectIDFromHex("64b6f7c2c0f1e84c0a1a9c01")
		if err != nil {
			t.Fatalf("failed to parse hex id: %v", err)
		}

		r := &bookingRepo{collection: br}
		bookings, err := r.FindByGuestId(ctx, guestID)
		if err != nil {
			t.Fatalf("FindByGuestId() unexpected error: %v", err)
		}

		if len(bookings) != 1 {
			t.Fatalf("FindByGuestId() expected 1 booking, got %d", len(bookings))
		}
		if bookings[0].CottageName != "Lake House" {
			t.Fatalf("FindByGuestId() unexpected cottage name: %+v", bookings[0])
		}
	})
}

func Test_bookingRepo_FindByGuestId_ReturnsEmptyWhenNone(t *testing.T) {
	setupAndRun("bookingRepo_FindByGuestId returns empty slice when none", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &bookingRepo{collection: br}
		bookings, err := r.FindByGuestId(ctx, primitive.NewObjectID())
		if err != nil {
			t.Fatalf("FindByGuestId() unexpected error: %v", err)
		}
		if len(bookings) != 0 {
			t.Fatalf("FindByGuestId() expected no bookings, got %d", len(bookings))
		}
	})
}

func Test_bookingRepo_FindByGuestId_ValidatesInput(t *testing.T) {
	setupAndRun("bookingRepo_FindByGuestId validates input", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &bookingRepo{collection: br}
		_, err := r.FindByGuestId(ctx, primitive.NilObjectID)

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("FindByGuestId() expected validation error, got %v", err)
		}
	})
}
