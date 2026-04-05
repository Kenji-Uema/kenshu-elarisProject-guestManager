package documents

import (
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Booking struct {
	Id             bson.ObjectID      `bson:"_id,omitempty"`
	MainGuest      bson.ObjectID      `bson:"main_guest"`
	NumberOfGuests int                `bson:"number_of_guests"`
	StayPeriod     Period             `bson:"stay_period"`
	CottageName    string             `bson:"cottage_name"`
	Status         enum.BookingStatus `bson:"status"`
}

type Period struct {
	CheckIn  time.Time `bson:"check_in"`
	CheckOut time.Time `bson:"check_out"`
}
