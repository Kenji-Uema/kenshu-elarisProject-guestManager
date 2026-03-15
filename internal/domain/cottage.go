package domain

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Cottage struct {
	Id            bson.ObjectID
	Name          string
	View          string
	Cleaned       bool
	Details       CottageDetails
	Photos        []string
	PricePerNight float32
	Bookings      []bson.ObjectID
	CurrentGuest  bson.ObjectID
}

type CottageDetails struct {
	Description          string
	View                 string
	FurnitureDescription string
	BathroomDescription  string
	AmenitiesDescription string
}
