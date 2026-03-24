package mdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Test_cottageRepo_UpdateCurrentGuest_UpdatesValue(t *testing.T) {
	setupAndRun("cottageRepo_UpdateCurrentGuest updates value", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		newGuestID := bson.NewObjectID()
		r := &cottageRepo{collection: ct}

		if err := r.UpdateCurrentGuest(ctx, "Lake House", newGuestID); err != nil {
			t.Fatalf("UpdateCurrentGuest() unexpected error: %v", err)
		}

		var updated map[string]any
		if err := ct.FindOne(ctx, bson.M{"name": "Lake House"}).Decode(&updated); err != nil {
			t.Fatalf("failed to load updated cottage: %v", err)
		}

		idHex := updated["current_guest"].(bson.ObjectID).Hex()
		if idHex != newGuestID.Hex() {
			t.Fatalf("UpdateCurrentGuest() did not persist current_guest change, got %s want %s", idHex, newGuestID.Hex())
		}
	})
}

func Test_cottageRepo_UpdateCurrentGuest_NotFound(t *testing.T) {
	setupAndRun("cottageRepo_UpdateCurrentGuest not found", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &cottageRepo{collection: ct}
		err := r.UpdateCurrentGuest(ctx, "Missing Cottage", bson.NewObjectID())

		var notFound *dbErrors.ErrCottageDoesNotExist
		if !errors.As(err, &notFound) {
			t.Fatalf("UpdateCurrentGuest() expected ErrCottageDoesNotExist, got %v", err)
		}
	})
}

func Test_cottageRepo_UpdateCurrentGuest_ValidatesInput(t *testing.T) {
	setupAndRun("cottageRepo_UpdateCurrentGuest validates input", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &cottageRepo{collection: ct}
		_, err := bson.ObjectIDFromHex("64b6f7c2c0f1e84c0a1a9c01") // ensure hex parse is covered
		if err != nil {
			t.Fatalf("failed to parse hex id: %v", err)
		}

		err = r.UpdateCurrentGuest(ctx, "", bson.NewObjectID())
		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("UpdateCurrentGuest() expected validation error for roomName, got %v", err)
		}

		err = r.UpdateCurrentGuest(ctx, "Lake House", bson.NilObjectID)
		if !errors.As(err, &validationErr) {
			t.Fatalf("UpdateCurrentGuest() expected validation error for guestId, got %v", err)
		}
	})
}

func Test_cottageRepo_RemovePastBooking_RemovesBookingAndClearsCurrentGuest(t *testing.T) {
	setupAndRun("cottageRepo_RemovePastBooking removes booking and clears current guest", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &cottageRepo{collection: ct}
		bookingID := bson.ObjectID{0x64, 0xb6, 0xf7, 0xc2, 0xc0, 0xf1, 0xe8, 0x4c, 0x0a, 0x1a, 0x9d, 0x01}

		if err := r.RemovePastBooking(ctx, bookingID); err != nil {
			t.Fatalf("RemovePastBooking() unexpected error: %v", err)
		}

		var updated map[string]any
		if err := ct.FindOne(ctx, bson.M{"name": "Lake House"}).Decode(&updated); err != nil {
			t.Fatalf("failed to load updated cottage: %v", err)
		}

		bookings, ok := updated["bookings"].(bson.A)
		if !ok {
			t.Fatalf("RemovePastBooking() bookings type = %T, want bson.A", updated["bookings"])
		}
		if len(bookings) != 0 {
			t.Fatalf("RemovePastBooking() bookings = %+v, want empty", bookings)
		}

		currentGuest := updated["current_guest"].(bson.ObjectID)
		if currentGuest != bson.NilObjectID {
			t.Fatalf("RemovePastBooking() current_guest = %s, want nil object id", currentGuest.Hex())
		}
	})
}

func Test_cottageRepo_RemovePastBooking_NotFound(t *testing.T) {
	setupAndRun("cottageRepo_RemovePastBooking not found", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &cottageRepo{collection: ct}
		err := r.RemovePastBooking(ctx, bson.NewObjectID())

		var notFound *dbErrors.ErrCottageDoesNotExist
		if !errors.As(err, &notFound) {
			t.Fatalf("RemovePastBooking() expected ErrCottageDoesNotExist, got %v", err)
		}
	})
}

func Test_cottageRepo_RemovePastBooking_ValidatesInput(t *testing.T) {
	setupAndRun("cottageRepo_RemovePastBooking validates input", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &cottageRepo{collection: ct}
		err := r.RemovePastBooking(ctx, bson.NilObjectID)

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("RemovePastBooking() expected validation error for bookingId, got %v", err)
		}
	})
}
