package domain

import (
	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Booking struct {
	mainGuest      bson.ObjectID
	numberOfGuests int
	stayPeriod     Period
	cottageName    string
	status         string
}

func NewBooking(mainGuest bson.ObjectID, numberOfGuests int, stayPeriod Period, cottageName string, status string) (Booking, error) {
	if err := validation.New().PositiveValue("numberOfGuests", numberOfGuests).
		NotZeroValue("stayPeriod", stayPeriod).
		NotBlank("cottageName", cottageName).
		NotBlank("status", status).Validate(); err != nil {
		return Booking{}, err
	}

	return Booking{
		mainGuest:      mainGuest,
		numberOfGuests: numberOfGuests,
		stayPeriod:     stayPeriod,
		cottageName:    cottageName,
		status:         status,
	}, nil
}
