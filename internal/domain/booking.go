package domain

import (
	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Booking struct {
	MainGuest      bson.ObjectID
	NumberOfGuests int
	StayPeriod     Period
	CottageName    string
	Status         string
}

func NewBooking(mainGuest bson.ObjectID, numberOfGuests int, stayPeriod Period, cottageName string, status string) (Booking, error) {
	if err := validation.New().PositiveValue("NumberOfGuests", numberOfGuests).
		NotZeroValue("StayPeriod", stayPeriod).
		NotBlank("CottageName", cottageName).
		NotBlank("Status", status).Validate(); err != nil {
		return Booking{}, err
	}

	return Booking{
		MainGuest:      mainGuest,
		NumberOfGuests: numberOfGuests,
		StayPeriod:     stayPeriod,
		CottageName:    cottageName,
		Status:         status,
	}, nil
}

func (b *Booking) ToDto() dto.BookingDto {
	return dto.BookingDto{
		NumberOfGuests: b.NumberOfGuests,
		StayPeriod: dto.Period{
			Start: b.StayPeriod.Start,
			End:   b.StayPeriod.End,
		},
		CottageName: b.CottageName,
		Status:      b.Status,
	}
}
