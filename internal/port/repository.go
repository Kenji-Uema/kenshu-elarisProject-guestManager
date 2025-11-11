package port

import (
	"context"
	"guestManager/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type GuestRepo interface {
	Get(ctx context.Context, id primitive.ObjectID) (domain.Guest, error)
	Add(ctx context.Context, newGuest domain.Guest) (primitive.ObjectID, error)
	Update(ctx context.Context, id primitive.ObjectID, guest domain.Guest) (domain.Guest, error)
}
