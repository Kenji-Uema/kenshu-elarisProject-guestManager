package mdb

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/config"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/guestManager/internal/port"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type cottageRepo struct {
	collection *mongo.Collection
}

func NewCottageRepo(db *mongo.Database, config config.CottageCollectionConfig) port.CottageRepo {
	return &cottageRepo{collection: db.Collection(config.Name)}
}

func (r *cottageRepo) UpdateCurrentGuest(ctx context.Context, roomName string, guestId bson.ObjectID) error {
	if err := validation.New().NotBlank("roomName", roomName).NotNilObjectID("guestId", guestId).Validate(); err != nil {
		return err
	}

	filter := bson.M{"name": roomName}
	update := bson.M{"$set": bson.M{"current_guest": guestId}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%w: could not update current guest for the room, roomName=%s guestId=%s: %v",
			dbErrors.ErrCottageRepo, roomName, guestId.Hex(), err)
	}

	if result.MatchedCount == 0 {
		return &dbErrors.ErrCottageDoesNotExist{CottageName: roomName}
	}

	return nil
}
