package documents

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Guest struct {
	Id         bson.ObjectID `bson:"_id,omitempty"`
	DocumentId string        `bson:"document_id"`
	GivenNames string        `bson:"given_names"`
	Surname    string        `bson:"surname"`
	Email      string        `bson:"email"`
}
