package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Booking struct {
	Id             primitive.ObjectID `bson:"_id,omitempty"`
	MainGuest      primitive.ObjectID `bson:"main_guest"`
	NumberOfGuests int                `bson:"number_of_guests"`
	StayPeriod     Period             `bson:"stay_period"`
	CottageName    string             `bson:"cottage_name"`
	Status         string             `bson:"status"`
}
