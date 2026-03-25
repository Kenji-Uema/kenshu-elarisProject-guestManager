package mdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func Test_guestRepo_GetById_NotFound(t *testing.T) {
	setupAndRun("guestRepo_GetById user not found", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &guestRepo{collection: gr}
		_, err := r.GetById(ctx, bson.NewObjectID())

		var errGuestNotFound *dbErrors.ErrGuestDoesNotExist
		if !errors.As(err, &errGuestNotFound) {
			t.Fatalf("GetById() expected ErrGuestDoesNotExist, got %v", err)
		}
	})
}

func Test_guestRepo_GetById_ValidatesId(t *testing.T) {
	setupAndRun("guestRepo_GetById validates id", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &guestRepo{collection: gr}
		_, err := r.GetById(ctx, bson.NilObjectID)

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("GetById() expected validation error, got %v", err)
		}
	})
}

func Test_guestRepo_GetById_ReturnsGuest(t *testing.T) {
	setupAndRun("guestRepo_GetById returns guest", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		id, err := bson.ObjectIDFromHex("64b6f7c2c0f1e84c0a1a9c01")
		if err != nil {
			t.Fatalf("failed to parse hex id: %v", err)
		}

		r := &guestRepo{collection: gr}
		got, err := r.GetById(ctx, id)
		if err != nil {
			t.Fatalf("GetById() unexpected error: %v", err)
		}

		if got.DocumentId != "ID-001" || got.GivenNames != "Alexandra Marie" || got.Surname != "Lopez" || got.BillingAddress != "Mountain Road 11" {
			t.Fatalf("GetById() returned unexpected guest: %+v", got)
		}
	})
}

func Test_guestRepo_GetByDocument_ValidatesInput(t *testing.T) {
	setupAndRun("guestRepo_GetByDocument validates input", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &guestRepo{collection: gr}
		_, err := r.GetByDocument(ctx, "")

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("GetByDocument() expected validation error, got %v", err)
		}
	})
}

func Test_guestRepo_GetByDocument_ReturnsGuest(t *testing.T) {
	setupAndRun("guestRepo_GetByDocument returns guest", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &guestRepo{collection: gr}
		got, err := r.GetByDocument(ctx, "ID-002")
		if err != nil {
			t.Fatalf("GetByDocument() unexpected error: %v", err)
		}

		if got.GivenNames != "Marcus" || got.Surname != "Nguyen" || got.BillingAddress != "City Avenue 21" {
			t.Fatalf("GetByDocument() returned unexpected guest: %+v", got)
		}
	})
}

func Test_guestRepo_GetByDocument_NotFound(t *testing.T) {
	setupAndRun("guestRepo_GetByDocument user not found", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &guestRepo{collection: gr}
		_, err := r.GetByDocument(ctx, "missing-id")

		var errGuestNotFound *dbErrors.ErrGuestDoesNotExist
		if !errors.As(err, &errGuestNotFound) {
			t.Fatalf("GetByDocument() expected ErrGuestDoesNotExist, got %v", err)
		}
	})
}

func Test_guestRepo_Add_InsertsGuest(t *testing.T) {
	setupAndRun("guestRepo_Add inserts guest", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &guestRepo{collection: gr}
		newGuest := documents.Guest{
			DocumentId:     "ID-999",
			GivenNames:     "Test",
			Surname:        "User",
			Email:          "test.user@example.com",
			BillingAddress: "Harbor Street 9",
		}

		insertedID, err := r.Add(ctx, newGuest)
		if err != nil {
			t.Fatalf("Add() unexpected error: %v", err)
		}
		if insertedID == bson.NilObjectID {
			t.Fatalf("Add() returned nil object ID")
		}

		inserted, err := r.GetById(ctx, insertedID)
		if err != nil {
			t.Fatalf("GetById() after add unexpected error: %v", err)
		}

		if inserted.DocumentId != newGuest.DocumentId || inserted.Email != newGuest.Email || inserted.BillingAddress != newGuest.BillingAddress {
			t.Fatalf("Add() stored unexpected guest: %+v", inserted)
		}
	})
}

func Test_guestRepo_Add_ValidatesInput(t *testing.T) {
	setupAndRun("guestRepo_Add validates input", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &guestRepo{collection: gr}
		_, err := r.Add(ctx, documents.Guest{})

		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("Add() expected validation error, got %v", err)
		}
	})
}

func Test_guestRepo_Update_UpdatesFields(t *testing.T) {
	setupAndRun("guestRepo_Update updates guest", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		id, err := bson.ObjectIDFromHex("64b6f7c2c0f1e84c0a1a9c02")
		if err != nil {
			t.Fatalf("failed to parse hex id: %v", err)
		}

		r := &guestRepo{collection: gr}
		updatedGuest := documents.Guest{
			DocumentId:     "ID-002-UPDATED",
			GivenNames:     "Marcus Aurelius",
			Surname:        "Nguyen",
			Email:          "marcus.aurelius@example.com",
			BillingAddress: "City Avenue 21",
		}

		got, err := r.Update(ctx, id, updatedGuest)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}

		if got.DocumentId != updatedGuest.DocumentId || got.Email != updatedGuest.Email || got.GivenNames != updatedGuest.GivenNames || got.BillingAddress != updatedGuest.BillingAddress {
			t.Fatalf("Update() returned unexpected guest: %+v", got)
		}

		reloaded, err := r.GetById(ctx, id)
		if err != nil {
			t.Fatalf("GetById() after update unexpected error: %v", err)
		}
		if reloaded.DocumentId != updatedGuest.DocumentId || reloaded.Email != updatedGuest.Email || reloaded.BillingAddress != updatedGuest.BillingAddress {
			t.Fatalf("Update() did not persist changes: %+v", reloaded)
		}
	})
}

func Test_guestRepo_Update_NotFound(t *testing.T) {
	setupAndRun("guestRepo_Update not found", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &guestRepo{collection: gr}
		updatedGuest := documents.Guest{
			DocumentId:     "ID-404",
			GivenNames:     "Ghost",
			Surname:        "Person",
			Email:          "ghost.person@example.com",
			BillingAddress: "Nowhere 0",
		}

		_, err := r.Update(ctx, bson.NewObjectID(), updatedGuest)

		var errGuestNotFound *dbErrors.ErrGuestDoesNotExist
		if !errors.As(err, &errGuestNotFound) {
			t.Fatalf("Update() expected ErrGuestDoesNotExist, got %v", err)
		}
	})
}

func Test_guestRepo_Update_NoChanges(t *testing.T) {
	setupAndRun("guestRepo_Update no changes", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		id, err := bson.ObjectIDFromHex("64b6f7c2c0f1e84c0a1a9c03")
		if err != nil {
			t.Fatalf("failed to parse hex id: %v", err)
		}

		r := &guestRepo{collection: gr}
		noChanges := documents.Guest{
			DocumentId:     "ID-003",
			GivenNames:     "Priya",
			Surname:        "Patel",
			Email:          "priya.patel@example.com",
			BillingAddress: "Garden Lane 7",
		}

		got, err := r.Update(ctx, id, noChanges)
		if err != nil {
			t.Fatalf("Update() unexpected error: %v", err)
		}

		if got.DocumentId != noChanges.DocumentId || got.Email != noChanges.Email || got.BillingAddress != noChanges.BillingAddress {
			t.Fatalf("Update() returned unexpected guest: %+v", got)
		}
	})
}

func Test_guestRepo_Update_ValidatesInput(t *testing.T) {
	setupAndRun("guestRepo_Update validates input", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection, gr *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &guestRepo{collection: gr}

		_, err := r.Update(ctx, bson.NilObjectID, documents.Guest{})
		var validationErr *validationErrors.ErrValidationConstrain
		if !errors.As(err, &validationErr) {
			t.Fatalf("Update() expected validation error for id and guest, got %v", err)
		}
	})
}
