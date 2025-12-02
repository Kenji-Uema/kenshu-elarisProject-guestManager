package documents

import "go.mongodb.org/mongo-driver/bson/primitive"

type Cottage struct {
	Id            primitive.ObjectID   `bson:"_id,omitempty"`
	Name          string               `bson:"name"`
	View          string               `bson:"view"`
	Details       CottageDetails       `bson:"details"`
	Photos        []string             `bson:"photos"`
	PricePerNight float32              `bson:"price_per_night"`
	Bookings      []primitive.ObjectID `bson:"bookings"`
	CurrentGuest  primitive.ObjectID   `bson:"current_guest"`
}

type CottageDetails struct {
	Description          string `bson:"description"`
	View                 string `bson:"view"`
	FurnitureDescription string `bson:"furniture_description"`
	BathroomDescription  string `bson:"bathroom_description"`
	AmenitiesDescription string `bson:"amenities_description"`
}
