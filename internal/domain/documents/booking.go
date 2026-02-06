package documents

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Booking struct {
	Id             bson.ObjectID `bson:"_id,omitempty"`
	MainGuest      bson.ObjectID `bson:"main_guest"`
	NumberOfGuests int           `bson:"number_of_guests"`
	StayPeriod     Period        `bson:"stay_period"`
	CottageName    string        `bson:"cottage_name"`
	Status         string        `bson:"status"`
}
