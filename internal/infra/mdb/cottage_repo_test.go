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
