package domain

import (
	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
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

func (b *Booking) ToDto() dto.BookingDto {
	return dto.BookingDto{
		NumberOfGuests: b.numberOfGuests,
		StayPeriod: dto.Period{
			Start: b.stayPeriod.start,
			End:   b.stayPeriod.end,
		},
		CottageName: b.cottageName,
		Status:      b.status,
	}
}
