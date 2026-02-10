package documents

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Guest struct {
	Id         bson.ObjectID `bson:"_id,omitempty"`
	DocumentId string        `bson:"document_id"`
	GivenNames string        `bson:"given_names"`
	Surname    string        `bson:"surname"`
	Email      string        `bson:"email"`
	CreatedAt  *time.Time    `bson:"created_at"`
	LastUpdate *time.Time    `bson:"last_update"`
}
