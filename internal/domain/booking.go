package domain

import (
	"guestManager/internal/domain/errors/appErrors"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Booking struct {
	id             primitive.ObjectID `bson:"_id,omitempty"`
	mainGuest      primitive.ObjectID `bson:"main_guest"`
	numberOfGuests int                `bson:"number_of_guests"`
	stayPeriod     Period             `bson:"stay_period"`
	cottageName    string             `bson:"cottage_name"`
	status         string             `bson:"status"`
}

func NewBooking(mainGuest primitive.ObjectID, numberOfGuests int, stayPeriod Period, cottageName string, status string) (Booking, error) {
	if mainGuest == primitive.NilObjectID {
		return Booking{}, &appErrors.ErrValidationConstrain{
			Field: "mainGuest", Message: "must not be null"}
	}
	if numberOfGuests < 1 {
		return Booking{}, &appErrors.ErrValidationConstrain{
			Field: "numberOgGuests", Message: "must be greater than 0"}
	}
	if stayPeriod == (Period{}) {
		return Booking{}, &appErrors.ErrValidationConstrain{
			Field: "period", Message: "must not be null"}
	}
	if strings.TrimSpace(cottageName) == "" {
		return Booking{}, &appErrors.ErrValidationConstrain{
			Field: "cottageName", Message: "must not be empty"}
	}
	if strings.TrimSpace(status) == "" {
		return Booking{}, &appErrors.ErrValidationConstrain{
			Field: "status", Message: "must not be empty"}
	}

	return Booking{
		mainGuest:      mainGuest,
		numberOfGuests: numberOfGuests,
		stayPeriod:     stayPeriod,
		cottageName:    cottageName,
		status:         status,
	}, nil
}
