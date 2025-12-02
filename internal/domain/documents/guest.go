package documents

import (
	"guestManager/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Guest struct {
	Id         primitive.ObjectID `bson:"_id,omitempty"`
	DocumentId string             `bson:"document_id"`
	GivenNames string             `bson:"given_names"`
	Surname    string             `bson:"surname"`
	Email      string             `bson:"email"`
}

func (g *Guest) ToDomain() (domain.Guest, error) {
	return domain.NewGuest(
		g.Id, g.DocumentId, g.GivenNames, g.Surname, g.Email)
}
