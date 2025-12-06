package domain

import (
	"guestManager/internal/app/validation"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Booking struct {
	mainGuest      primitive.ObjectID
	numberOfGuests int
	stayPeriod     Period
	cottageName    string
	status         string
}

func NewBooking(mainGuest primitive.ObjectID, numberOfGuests int, stayPeriod Period, cottageName string, status string) (Booking, error) {
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
