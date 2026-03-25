package domain

import (
	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Booking struct {
	Id             bson.ObjectID
	MainGuest      bson.ObjectID
	NumberOfGuests int
	StayPeriod     Period
	CottageName    string
	Status         enum.BookingStatus
}

func NewBooking(id bson.ObjectID, mainGuest bson.ObjectID, numberOfGuests int, stayPeriod Period, cottageName string, status enum.BookingStatus) (Booking, error) {
	if err := validation.New().NotNilObjectID("Id", id).
		PositiveValue("NumberOfGuests", numberOfGuests).
		NotZeroValue("StayPeriod", stayPeriod).
		NotBlank("CottageName", cottageName).
		NotBlank("Status", string(status)).Validate(); err != nil {
		return Booking{}, err
	}

	bookingStatus := status

	return Booking{
		Id:             id,
		MainGuest:      mainGuest,
		NumberOfGuests: numberOfGuests,
		StayPeriod:     stayPeriod,
		CottageName:    cottageName,
		Status:         bookingStatus,
	}, nil
}

func (b *Booking) ToDto() dto.BookingDto {
	return dto.BookingDto{
		NumberOfGuests: b.NumberOfGuests,
		StayPeriod: dto.Period{
			CheckIn:  b.StayPeriod.CheckIn,
			CheckOut: b.StayPeriod.CheckOut,
		},
		CottageName: b.CottageName,
		Status:      string(b.Status),
	}
}
