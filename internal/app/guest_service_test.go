package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/validationErrors"
	mdbfakes "github.com/Kenji-Uema/guestManager/internal/infra/mdb/fakes"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var guestID = bson.NewObjectID()
var guestFixedTime = time.Unix(1_700_000_000, 0).UTC()
var guestDoc = documents.Guest{
	Id:             guestID,
	DocumentId:     "1234567890",
	GivenNames:     "John",
	Surname:        "Smith",
	Email:          "test@test.com",
	BillingAddress: "Billing Street 1",
	CreatedAt:      &guestFixedTime,
	LastUpdate:     &guestFixedTime,
}

func newFakeGuestRepo(seed map[bson.ObjectID]documents.Guest) *mdbfakes.FakeGuestRepo {
	store := make(map[bson.ObjectID]documents.Guest, len(seed))
	for id, guest := range seed {
		store[id] = guest
	}

	return &mdbfakes.FakeGuestRepo{
		GetByIdFn: func(ctx context.Context, id bson.ObjectID) (documents.Guest, error) {
			guest, ok := store[id]
			if !ok {
				return documents.Guest{}, &dbErrors.ErrGuestDoesNotExist{Id: id}
			}
			return guest, nil
		},
		GetByDocumentFn: func(ctx context.Context, documentId string) (documents.Guest, error) {
			for _, guest := range store {
				if guest.DocumentId == documentId {
					return guest, nil
				}
			}
			return documents.Guest{}, &dbErrors.ErrGuestDoesNotExist{DocumentId: documentId}
		},
		AddFn: func(ctx context.Context, newGuest documents.Guest) (bson.ObjectID, error) {
			store[newGuest.Id] = newGuest
			return newGuest.Id, nil
		},
		UpdateFn: func(ctx context.Context, id bson.ObjectID, guest documents.Guest) (documents.Guest, error) {
			if _, ok := store[id]; !ok {
				return documents.Guest{}, &dbErrors.ErrGuestDoesNotExist{Id: id}
			}
			store[id] = guest
			return guest, nil
		},
	}
}

func TestGuestServiceAdd(t *testing.T) {
	repo := newFakeGuestRepo(map[bson.ObjectID]documents.Guest{guestID: guestDoc})
	service, err := NewGuestService(repo, &mdbfakes.FakeBookingRepo{})
	if err != nil {
		t.Fatalf("NewGuestService() error = %v", err)
	}

	newGuestID := bson.NewObjectID()
	now := time.Now().UTC()
	guest, err := domain.NewGuest(newGuestID, "0987654321", "Alice", "Doe", "alice@example.com", "Billing Street 2", &now, &now)
	if err != nil {
		t.Fatalf("NewGuest() error = %v", err)
	}

	gotID, err := service.Add(t.Context(), guest)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if gotID != newGuestID {
		t.Fatalf("Add() id = %s, want %s", gotID.Hex(), newGuestID.Hex())
	}

	stored, err := repo.GetById(t.Context(), newGuestID)
	if err != nil {
		t.Fatalf("repo.GetById() error = %v", err)
	}
	if !reflect.DeepEqual(stored, guest.ToMongoDoc()) {
		t.Fatalf("repo stored guest = %#v, want %#v", stored, guest.ToMongoDoc())
	}
}

func TestGuestServiceGetByDocument(t *testing.T) {
	wantGuest, err := domain.NewGuest(
		guestID,
		guestDoc.DocumentId,
		guestDoc.GivenNames,
		guestDoc.Surname,
		guestDoc.Email,
		guestDoc.BillingAddress,
		guestDoc.CreatedAt,
		guestDoc.LastUpdate,
	)
	if err != nil {
		t.Fatalf("NewGuest() error = %v", err)
	}

	t.Run("returns guest when document exists", func(t *testing.T) {
		service, err := NewGuestService(newFakeGuestRepo(map[bson.ObjectID]documents.Guest{guestID: guestDoc}), &mdbfakes.FakeBookingRepo{})
		if err != nil {
			t.Fatalf("NewGuestService() error = %v", err)
		}

		got, err := service.GetByDocument(t.Context(), guestDoc.DocumentId)
		if err != nil {
			t.Fatalf("GetByDocument() error = %v", err)
		}
		if !reflect.DeepEqual(got, wantGuest) {
			t.Fatalf("GetByDocument() = %#v, want %#v", got, wantGuest)
		}
	})

	t.Run("returns not found error when document does not exist", func(t *testing.T) {
		service, err := NewGuestService(newFakeGuestRepo(map[bson.ObjectID]documents.Guest{guestID: guestDoc}), &mdbfakes.FakeBookingRepo{})
		if err != nil {
			t.Fatalf("NewGuestService() error = %v", err)
		}

		got, err := service.GetByDocument(t.Context(), "does-not-exist")
		if got != (domain.Guest{}) {
			t.Fatalf("GetByDocument() guest = %#v, want zero value", got)
		}

		var target *dbErrors.ErrGuestDoesNotExist
		if !errors.As(err, &target) {
			t.Fatalf("GetByDocument() error = %v, want ErrGuestDoesNotExist", err)
		}
	})
}

func TestGuestServiceGetByID(t *testing.T) {
	wantGuest, err := domain.NewGuest(
		guestID,
		guestDoc.DocumentId,
		guestDoc.GivenNames,
		guestDoc.Surname,
		guestDoc.Email,
		guestDoc.BillingAddress,
		guestDoc.CreatedAt,
		guestDoc.LastUpdate,
	)
	if err != nil {
		t.Fatalf("NewGuest() error = %v", err)
	}

	t.Run("returns guest when id exists", func(t *testing.T) {
		service, err := NewGuestService(newFakeGuestRepo(map[bson.ObjectID]documents.Guest{guestID: guestDoc}), &mdbfakes.FakeBookingRepo{})
		if err != nil {
			t.Fatalf("NewGuestService() error = %v", err)
		}

		got, err := service.GetById(t.Context(), guestID)
		if err != nil {
			t.Fatalf("GetById() error = %v", err)
		}
		if !reflect.DeepEqual(got, wantGuest) {
			t.Fatalf("GetById() = %#v, want %#v", got, wantGuest)
		}
	})

	t.Run("returns not found error when id does not exist", func(t *testing.T) {
		service, err := NewGuestService(newFakeGuestRepo(map[bson.ObjectID]documents.Guest{guestID: guestDoc}), &mdbfakes.FakeBookingRepo{})
		if err != nil {
			t.Fatalf("NewGuestService() error = %v", err)
		}

		got, err := service.GetById(t.Context(), bson.NewObjectID())
		if got != (domain.Guest{}) {
			t.Fatalf("GetById() guest = %#v, want zero value", got)
		}

		var target *dbErrors.ErrGuestDoesNotExist
		if !errors.As(err, &target) {
			t.Fatalf("GetById() error = %v, want ErrGuestDoesNotExist", err)
		}
	})

	t.Run("returns validation error when stored guest is invalid", func(t *testing.T) {
		badID := bson.NewObjectID()
		service, err := NewGuestService(newFakeGuestRepo(map[bson.ObjectID]documents.Guest{
			badID: {
				Id:             bson.NilObjectID,
				DocumentId:     "invalid",
				GivenNames:     "John",
				Surname:        "Smith",
				Email:          "test@test.com",
				BillingAddress: "Billing Street 1",
			},
		}), &mdbfakes.FakeBookingRepo{})
		if err != nil {
			t.Fatalf("NewGuestService() error = %v", err)
		}

		got, err := service.GetById(t.Context(), badID)
		if got != (domain.Guest{}) {
			t.Fatalf("GetById() guest = %#v, want zero value", got)
		}

		var target *validationErrors.ErrValidationConstrain
		if !errors.As(err, &target) {
			t.Fatalf("GetById() error = %v, want ErrValidationConstrain", err)
		}
	})
}

func TestGuestServiceUpdate(t *testing.T) {
	t.Run("updates guest and returns updated domain object", func(t *testing.T) {
		repo := newFakeGuestRepo(map[bson.ObjectID]documents.Guest{guestID: guestDoc})
		service, err := NewGuestService(repo, &mdbfakes.FakeBookingRepo{})
		if err != nil {
			t.Fatalf("NewGuestService() error = %v", err)
		}

		now := guestFixedTime.Add(time.Hour)
		updatedGuest, err := domain.NewGuest(
			guestID,
			"1234567890",
			"Johnny",
			"Doe",
			"johnny@test.com",
			"Billing Street 1",
			&guestFixedTime,
			&now,
		)
		if err != nil {
			t.Fatalf("NewGuest() error = %v", err)
		}

		got, err := service.Update(t.Context(), guestID, updatedGuest)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if !reflect.DeepEqual(got, updatedGuest) {
			t.Fatalf("Update() = %#v, want %#v", got, updatedGuest)
		}

		stored, err := repo.GetById(t.Context(), guestID)
		if err != nil {
			t.Fatalf("repo.GetById() error = %v", err)
		}
		if !reflect.DeepEqual(stored, updatedGuest.ToMongoDoc()) {
			t.Fatalf("repo stored guest = %#v, want %#v", stored, updatedGuest.ToMongoDoc())
		}
	})

	t.Run("returns validation error when updated guest is invalid", func(t *testing.T) {
		repo := newFakeGuestRepo(map[bson.ObjectID]documents.Guest{guestID: guestDoc})
		service, err := NewGuestService(repo, &mdbfakes.FakeBookingRepo{})
		if err != nil {
			t.Fatalf("NewGuestService() error = %v", err)
		}

		got, err := service.Update(t.Context(), guestID, domain.Guest{})
		if got != (domain.Guest{}) {
			t.Fatalf("Update() guest = %#v, want zero value", got)
		}

		var target *validationErrors.ErrValidationConstrain
		if !errors.As(err, &target) {
			t.Fatalf("Update() error = %v, want ErrValidationConstrain", err)
		}
	})
}
