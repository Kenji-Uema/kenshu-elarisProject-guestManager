package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type Guest struct {
	Id         primitive.ObjectID `bson:"_id,omitempty"`
	DocumentId string             `bson:"document_id"`
	GivenNames string             `bson:"given_names"`
	Surname    string             `bson:"surname"`
	Email      string             `bson:"email"`
}
