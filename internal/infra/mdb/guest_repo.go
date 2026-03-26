package mdb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/guestManager/internal/port"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type guestRepo struct {
	collection *mongo.Collection
}

func NewGuestRepo(db *mongo.Database, collectionName string) port.GuestRepo {
	return &guestRepo{collection: db.Collection(collectionName)}
}

func (g *guestRepo) GetById(ctx context.Context, id bson.ObjectID) (documents.Guest, error) {
	if err := validation.New().NotNilObjectID("id", id).Validate(); err != nil {
		return documents.Guest{}, err
	}

	result := g.collection.FindOne(ctx, bson.M{"_id": id})

	var guest documents.Guest
	if err := result.Decode(&guest); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return documents.Guest{}, &dbErrors.ErrGuestDoesNotExist{Id: id}
		}
		return documents.Guest{}, fmt.Errorf("%w: search by Id failed; could not decode guest; guestId=%s: %v",
			dbErrors.ErrGuestRepo, id.Hex(), err)
	}
	return guest, nil
}

func (g *guestRepo) GetByDocument(ctx context.Context, documentId string) (documents.Guest, error) {
	if err := validation.New().NotBlank("documentId", documentId).Validate(); err != nil {
		return documents.Guest{}, err
	}

	result := g.collection.FindOne(ctx, bson.M{"document_id": documentId})

	var guest documents.Guest
	if err := result.Decode(&guest); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return documents.Guest{}, &dbErrors.ErrGuestDoesNotExist{DocumentId: documentId}
		}
		return documents.Guest{}, fmt.Errorf("%w: searchBy docoumentId failed; could not decode guest; documentId=%s: %v",
			dbErrors.ErrGuestRepo, documentId, err)
	}
	return guest, nil
}

func (g *guestRepo) Add(ctx context.Context, newGuest documents.Guest) (bson.ObjectID, error) {
	if err := validation.New().NotZeroValue("newGuest", newGuest).Validate(); err != nil {
		return bson.NilObjectID, err
	}

	result, err := g.collection.InsertOne(ctx, newGuest)

	if err != nil {
		slog.ErrorContext(ctx, "failed to insert guest", "error", err)
		return bson.NilObjectID, fmt.Errorf("%w: add new guest failed; guest=%v: %v",
			dbErrors.ErrGuestRepo, newGuest, err)
	}

	return result.InsertedID.(bson.ObjectID), nil
}

func (g *guestRepo) Update(ctx context.Context, id bson.ObjectID, updatedGuest documents.Guest) (documents.Guest, error) {
	if err := validation.New().NotNilObjectID("id", id).NotZeroValue("updatedGuest", updatedGuest).Validate(); err != nil {
		return documents.Guest{}, err
	}

	filter := bson.M{"_id": id}

	var existingGuest documents.Guest
	if err := g.collection.FindOne(ctx, filter).Decode(&existingGuest); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return documents.Guest{}, &dbErrors.ErrGuestDoesNotExist{Id: id}
		}
		return documents.Guest{}, fmt.Errorf("%w: update guest failed; guestId=%s: %v",
			dbErrors.ErrGuestRepo, id.Hex(), err)
	}

	updateFields := bson.M{}

	setStringIfChanged(updateFields, "document_id", &existingGuest.DocumentId, updatedGuest.DocumentId)
	setStringIfChanged(updateFields, "given_names", &existingGuest.GivenNames, updatedGuest.GivenNames)
	setStringIfChanged(updateFields, "surname", &existingGuest.Surname, updatedGuest.Surname)
	setStringIfChanged(updateFields, "email", &existingGuest.Email, updatedGuest.Email)
	setStringIfChanged(updateFields, "billing_address", &existingGuest.BillingAddress, updatedGuest.BillingAddress)

	if len(updateFields) == 0 {
		return existingGuest, nil
	}

	if _, err := g.collection.UpdateOne(ctx, filter, bson.M{"$set": updateFields}); err != nil {
		return documents.Guest{}, fmt.Errorf("%w: update guest failed; guestId=%s: %v",
			dbErrors.ErrGuestRepo, id.Hex(), err)
	}

	return existingGuest, nil
}

func setStringIfChanged(m bson.M, key string, current *string, newVal string) {
	if newVal != "" && newVal != *current {
		m[key] = newVal
		*current = newVal
	}
}
