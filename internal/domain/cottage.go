package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Cottage struct {
	Id            primitive.ObjectID
	Name          string
	View          string
	Details       CottageDetails
	Photos        []string
	PricePerNight float32
	Bookings      []primitive.ObjectID
	CurrentGuest  primitive.ObjectID
}

type CottageDetails struct {
	Description          string
	View                 string
	FurnitureDescription string
	BathroomDescription  string
	AmenitiesDescription string
}
