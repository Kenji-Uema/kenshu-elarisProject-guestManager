package mdb

import (
	"context"
	"errors"
	"guestManager/internal/domain/errors/dbErrors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func Test_guestRepo_GetById(t *testing.T) {
	setupAndRun("guestRepo_GetById user not found", t, func(t *testing.T, ct *mongo.Collection, br *mongo.Collection) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		r := &guestRepo{collection: ct}
		_, err := r.GetById(ctx, primitive.NewObjectID())

		var errGuestNotFound *dbErrors.ErrGuestDoesNotExist
		if !errors.As(err, &errGuestNotFound) {
			t.Fatalf("GetAll() expected to throw an error with message 'guest not found', got %s", err.Error())
		}
	})
}
