package mdb

import (
	"context"
	"errors"
	"guestManager/internal/config"
	"guestManager/internal/domain"
	"guestManager/internal/port"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type guestRepo struct {
	collection *mongo.Collection
}

func NewGuestRepo(db *mongo.Database, config *config.GuestCollectionConfig) port.GuestRepo {
	return &guestRepo{collection: db.Collection(config.Name)}
}

func (g guestRepo) GetById(ctx context.Context, id primitive.ObjectID) (domain.Guest, error) {
	if id == primitive.NilObjectID {
		return domain.Guest{}, errors.New("guest id is required")
	}

	result := g.collection.FindOne(ctx, bson.M{"_id": id})

	var guest domain.Guest
	if err := result.Decode(&guest); err != nil {
		return domain.Guest{}, err
	}
	return guest, nil
}

func (g guestRepo) GetByDocument(ctx context.Context, documentId string) (domain.Guest, error) {
	result := g.collection.FindOne(ctx, bson.M{"document_id": documentId})

	var guest domain.Guest
	if err := result.Decode(&guest); err != nil {
		return domain.Guest{}, err
	}
	return guest, nil
}

func (g guestRepo) Add(ctx context.Context, newGuest domain.Guest) (primitive.ObjectID, error) {
	result, err := g.collection.InsertOne(ctx, newGuest)

	if err != nil {
		slog.Error("failed to insert guest", "error", err)
		return primitive.NilObjectID, err
	}

	return result.InsertedID.(primitive.ObjectID), nil
}

func (g guestRepo) Update(ctx context.Context, id primitive.ObjectID, updatedGuest domain.Guest) (domain.Guest, error) {
	if id == primitive.NilObjectID {
		return domain.Guest{}, errors.New("guest id is required for update")
	}

	var existingGuest domain.Guest
	if err := g.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&existingGuest); err != nil {
		return domain.Guest{}, err
	}

	updateFields := bson.M{}

	if updatedGuest.DocumentId != "" && updatedGuest.DocumentId != existingGuest.DocumentId {
		updateFields["document_id"] = updatedGuest.DocumentId
		existingGuest.DocumentId = updatedGuest.DocumentId
	}
	if updatedGuest.GivenNames != "" && updatedGuest.GivenNames != existingGuest.GivenNames {
		updateFields["given_names"] = updatedGuest.GivenNames
		existingGuest.GivenNames = updatedGuest.GivenNames
	}
	if updatedGuest.Surname != "" && updatedGuest.Surname != existingGuest.Surname {
		updateFields["surname"] = updatedGuest.Surname
		existingGuest.Surname = updatedGuest.Surname
	}
	if updatedGuest.Email != "" && updatedGuest.Email != existingGuest.Email {
		updateFields["email"] = updatedGuest.Email
		existingGuest.Email = updatedGuest.Email
	}

	if len(updateFields) == 0 {
		return existingGuest, nil
	}

	if _, err := g.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updateFields}); err != nil {
		return domain.Guest{}, err
	}

	return existingGuest, nil
}
