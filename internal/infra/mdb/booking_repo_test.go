package mdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Test_bookingRepo_FindByGuestId_ReturnsBookings(t *testing.T) {
	setupAndRun("bookingRepo_FindByGuestId returns bookings", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		guestID, err := bson.ObjectIDFromHex("64b6f7c2c0f1e84c0a1a9c01")
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
		bookings, err := r.FindByGuestId(ctx, bson.NewObjectID())
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
		_, err := r.FindByGuestId(ctx, bson.NilObjectID)

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("FindByGuestId() expected validation error, got %v", err)
		}
	})
}

func Test_bookingRepo_FindByGuestIdAndCheckIn_ReturnsBooking(t *testing.T) {
	setupAndRun("bookingRepo_FindByGuestIdAndCheckIn returns booking", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		guestID, err := bson.ObjectIDFromHex("64b6f7c2c0f1e84c0a1a9c01")
		if err != nil {
			t.Fatalf("failed to parse hex id: %v", err)
		}
		checkIn := time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC)

		r := &bookingRepo{collection: br}
		booking, err := r.FindByGuestIdAndCheckIn(ctx, guestID, checkIn)
		if err != nil {
			t.Fatalf("FindByGuestIdAndCheckIn() unexpected error: %v", err)
		}

		if booking.CottageName != "Lake House" {
			t.Fatalf("FindByGuestIdAndCheckIn() unexpected cottage name: %+v", booking)
		}
	})
}

func Test_bookingRepo_FindByGuestIdAndCheckIn_ReturnsErrorWhenNone(t *testing.T) {
	setupAndRun("bookingRepo_FindByGuestIdAndCheckIn returns error when none", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &bookingRepo{collection: br}
		_, err := r.FindByGuestIdAndCheckIn(ctx, bson.NewObjectID(), time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC))
		if err == nil {
			t.Fatal("FindByGuestIdAndCheckIn() expected error, got nil")
		}
	})
}

func Test_bookingRepo_FindByGuestIdAndCheckIn_ValidatesInput(t *testing.T) {
	setupAndRun("bookingRepo_FindByGuestIdAndCheckIn validates input", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &bookingRepo{collection: br}
		_, err := r.FindByGuestIdAndCheckIn(ctx, bson.NilObjectID, time.Time{})

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("FindByGuestIdAndCheckIn() expected validation error, got %v", err)
		}
	})
}

func Test_bookingRepo_FindByGuestIdAndBookingNumber_ReturnsBooking(t *testing.T) {
	setupAndRun("bookingRepo_FindByGuestIdAndBookingNumber returns booking", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		guestID, err := bson.ObjectIDFromHex("64b6f7c2c0f1e84c0a1a9c01")
		if err != nil {
			t.Fatalf("failed to parse hex guest id: %v", err)
		}

		r := &bookingRepo{collection: br}
		booking, err := r.FindByGuestIdAndBookingNumber(ctx, guestID, "64b6f7c2c0f1e84c0a1a9d01")
		if err != nil {
			t.Fatalf("FindByGuestIdAndBookingNumber() unexpected error: %v", err)
		}

		if booking.CottageName != "Lake House" {
			t.Fatalf("FindByGuestIdAndBookingNumber() unexpected cottage name: %+v", booking)
		}
	})
}

func Test_bookingRepo_FindByGuestIdAndBookingNumber_ReturnsErrorWhenNone(t *testing.T) {
	setupAndRun("bookingRepo_FindByGuestIdAndBookingNumber returns error when none", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &bookingRepo{collection: br}
		_, err := r.FindByGuestIdAndBookingNumber(ctx, bson.NewObjectID(), "64b6f7c2c0f1e84c0a1a9d01")
		if err == nil {
			t.Fatal("FindByGuestIdAndBookingNumber() expected error, got nil")
		}
	})
}

func Test_bookingRepo_FindByGuestIdAndBookingNumber_ValidatesInput(t *testing.T) {
	setupAndRun("bookingRepo_FindByGuestIdAndBookingNumber validates input", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &bookingRepo{collection: br}
		_, err := r.FindByGuestIdAndBookingNumber(ctx, bson.NilObjectID, "")

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("FindByGuestIdAndBookingNumber() expected validation error, got %v", err)
		}
	})
}

func Test_bookingRepo_FindBookingByCheckInDate_ReturnsBookings(t *testing.T) {
	setupAndRun("bookingRepo_FindBookingByCheckInDate returns bookings", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &bookingRepo{collection: br}
		bookings, err := r.FindByCheckInDate(ctx, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("FindByCheckInDate() unexpected error: %v", err)
		}

		if len(bookings) != 1 {
			t.Fatalf("FindByCheckInDate() expected 1 booking, got %d", len(bookings))
		}

		expectedID, err := bson.ObjectIDFromHex("64b6f7c2c0f1e84c0a1a9d01")
		if err != nil {
			t.Fatalf("failed to parse expected booking id: %v", err)
		}
		if bookings[0].Id != expectedID {
			t.Fatalf("FindByCheckInDate() expected id %s, got %s", expectedID.Hex(), bookings[0].Id.Hex())
		}
		if bookings[0].CottageName != "Lake House" {
			t.Fatalf("FindByCheckInDate() unexpected cottage name: %+v", bookings[0])
		}
	})
}

func Test_bookingRepo_FindBookingByCheckInDate_ReturnsEmptyWhenNone(t *testing.T) {
	setupAndRun("bookingRepo_FindBookingByCheckInDate returns empty when none", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &bookingRepo{collection: br}
		bookings, err := r.FindByCheckInDate(ctx, time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("FindByCheckInDate() unexpected error: %v", err)
		}

		if len(bookings) != 0 {
			t.Fatalf("FindByCheckInDate() expected empty result, got %d", len(bookings))
		}
	})
}

func Test_bookingRepo_FindBookingByCheckInDate_ValidatesInput(t *testing.T) {
	setupAndRun("bookingRepo_FindBookingByCheckInDate validates input", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &bookingRepo{collection: br}
		_, err := r.FindByCheckInDate(ctx, time.Time{})

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("FindByCheckInDate() expected validation error, got %v", err)
		}
	})
}

func Test_bookingRepo_UpdateStatus_UpdatesBooking(t *testing.T) {
	setupAndRun("bookingRepo_UpdateStatus updates booking", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		bookingID, err := bson.ObjectIDFromHex("64b6f7c2c0f1e84c0a1a9d01")
		if err != nil {
			t.Fatalf("failed to parse booking id: %v", err)
		}

		r := &bookingRepo{collection: br}
		if err := r.UpdateStatus(ctx, bookingID, enum.BookingStatusPast); err != nil {
			t.Fatalf("UpdateStatus() unexpected error: %v", err)
		}

		var updated bson.M
		if err := br.FindOne(ctx, bson.M{"_id": bookingID}).Decode(&updated); err != nil {
			t.Fatalf("FindOne() unexpected error: %v", err)
		}
		if updated["status"] != string(enum.BookingStatusPast) {
			t.Fatalf("UpdateStatus() status = %v, want %q", updated["status"], enum.BookingStatusPast)
		}
	})
}

func Test_bookingRepo_UpdateStatus_ValidatesInput(t *testing.T) {
	setupAndRun("bookingRepo_UpdateStatus validates input", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &bookingRepo{collection: br}
		err := r.UpdateStatus(ctx, bson.NilObjectID, "")

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("UpdateStatus() expected validation error, got %v", err)
		}
	})
}
