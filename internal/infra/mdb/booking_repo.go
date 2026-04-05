package mdb

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	"github.com/Kenji-Uema/guestManager/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type bookingRepo struct {
	collection *mongo.Collection
}

type bookingDocumentCurrent struct {
	Id             bson.ObjectID         `bson:"_id,omitempty"`
	MainGuest      bson.ObjectID         `bson:"main_guest"`
	NumberOfGuests int                   `bson:"number_of_guests"`
	StayPeriod     periodDocumentCurrent `bson:"stay_period"`
	CottageName    string                `bson:"cottage_name"`
	Status         enum.BookingStatus    `bson:"status"`
}

type periodDocumentCurrent struct {
	CheckIn  time.Time `bson:"check_in"`
	CheckOut time.Time `bson:"check_out"`
}

func NewBookingRepo(db *mongo.Database, collectionName string) port.BookingRepo {
	return &bookingRepo{collection: db.Collection(collectionName)}
}

func decodeBooking(raw bson.Raw) (documents.Booking, error) {
	var current bookingDocumentCurrent
	if err := bson.Unmarshal(raw, &current); err != nil {
		return documents.Booking{}, err
	}

	return documents.Booking{
		Id:             current.Id,
		MainGuest:      current.MainGuest,
		NumberOfGuests: current.NumberOfGuests,
		StayPeriod: documents.Period{
			CheckIn:  current.StayPeriod.CheckIn,
			CheckOut: current.StayPeriod.CheckOut,
		},
		CottageName: current.CottageName,
		Status:      current.Status,
	}, nil
}

func (b *bookingRepo) FindByGuestId(ctx context.Context, guestId bson.ObjectID) ([]documents.Booking, error) {
	if err := validation.New().NotNilObjectID("guestId", guestId).Validate(); err != nil {
		return nil, err
	}

	cursor, err := b.collection.Find(ctx, bson.M{"main_guest": guestId})
	if err != nil {
		return nil, fmt.Errorf("%w: could not find bookings for guestId=%s: %v",
			dbErrors.ErrBookingRepo, guestId.Hex(), err)
	}

	//goland:noinspection GoUnhandledErrorResult
	defer cursor.Close(ctx)

	var raws []bson.Raw
	if err := cursor.All(ctx, &raws); err != nil {
		return nil, fmt.Errorf("%w: could not find bookings for guestId=%s: %v",
			dbErrors.ErrBookingRepo, guestId.Hex(), err)
	}

	bookings := make([]documents.Booking, 0, len(raws))
	for _, raw := range raws {
		booking, err := decodeBooking(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: could not decode bookings for guestId=%s: %v",
				dbErrors.ErrBookingRepo, guestId.Hex(), err)
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

func (b *bookingRepo) FindByCheckInDate(ctx context.Context, date time.Time) ([]documents.Booking, error) {
	if err := validation.New().NotZeroValue("date", date).Validate(); err != nil {
		return nil, err
	}

	y, m, d := date.UTC().Date()
	startOfDay := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	nextDay := startOfDay.Add(24 * time.Hour)

	filter := bson.M{
		"stay_period.check_in": bson.M{
			"$gte": startOfDay,
			"$lt":  nextDay,
		},
	}

	cursor, err := b.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("%w: could not find bookings for checkInDate=%s: %v",
			dbErrors.ErrBookingRepo, startOfDay.Format(time.RFC3339), err)
	}

	//goland:noinspection GoUnhandledErrorResult
	defer cursor.Close(ctx)

	var raws []bson.Raw
	if err := cursor.All(ctx, &raws); err != nil {
		return nil, fmt.Errorf("%w: could not decode bookings for checkInDate=%s: %v",
			dbErrors.ErrBookingRepo, startOfDay.Format(time.RFC3339), err)
	}

	bookings := make([]documents.Booking, 0, len(raws))
	for _, raw := range raws {
		booking, err := decodeBooking(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: could not decode bookings for checkInDate=%s: %v",
				dbErrors.ErrBookingRepo, startOfDay.Format(time.RFC3339), err)
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

func (b *bookingRepo) FindByGuestIdAndCheckIn(ctx context.Context, guestId bson.ObjectID, checkIn time.Time) (documents.Booking, error) {
	if err := validation.New().
		NotNilObjectID("guestId", guestId).
		NotZeroValue("checkIn", checkIn).
		Validate(); err != nil {
		return documents.Booking{}, err
	}

	filter := bson.M{
		"main_guest":           guestId,
		"stay_period.check_in": checkIn,
	}

	var raw bson.Raw
	if err := b.collection.FindOne(ctx, filter).Decode(&raw); err != nil {
		return documents.Booking{}, fmt.Errorf("%w: could not find booking for guestId=%s and checkIn=%s: %v",
			dbErrors.ErrBookingRepo, guestId.Hex(), checkIn.UTC().Format(time.RFC3339), err)
	}

	booking, err := decodeBooking(raw)
	if err != nil {
		return documents.Booking{}, fmt.Errorf("%w: could not decode booking for guestId=%s and checkIn=%s: %v",
			dbErrors.ErrBookingRepo, guestId.Hex(), checkIn.UTC().Format(time.RFC3339), err)
	}

	return booking, nil
}

func (b *bookingRepo) FindByGuestIdAndBookingNumber(ctx context.Context, guestId bson.ObjectID, bookingNumber string) (documents.Booking, error) {
	if err := validation.New().
		NotNilObjectID("guestId", guestId).
		NotBlank("bookingNumber", bookingNumber).
		Validate(); err != nil {
		return documents.Booking{}, err
	}

	bookingID, err := bson.ObjectIDFromHex(bookingNumber)
	if err != nil {
		return documents.Booking{}, fmt.Errorf("%w: invalid booking number %q: %v",
			dbErrors.ErrBookingRepo, bookingNumber, err)
	}

	filter := bson.M{
		"_id":        bookingID,
		"main_guest": guestId,
	}

	var raw bson.Raw
	if err := b.collection.FindOne(ctx, filter).Decode(&raw); err != nil {
		return documents.Booking{}, fmt.Errorf("%w: could not find booking for guestId=%s and bookingNumber=%s: %v",
			dbErrors.ErrBookingRepo, guestId.Hex(), bookingNumber, err)
	}

	booking, err := decodeBooking(raw)
	if err != nil {
		return documents.Booking{}, fmt.Errorf("%w: could not decode booking for guestId=%s and bookingNumber=%s: %v",
			dbErrors.ErrBookingRepo, guestId.Hex(), bookingNumber, err)
	}

	return booking, nil
}

func (b *bookingRepo) UpdateStatus(ctx context.Context, bookingId bson.ObjectID, status enum.BookingStatus) error {
	if err := validation.New().
		NotNilObjectID("bookingId", bookingId).
		NotBlank("status", string(status)).
		Validate(); err != nil {
		return err
	}

	result, err := b.collection.UpdateOne(ctx, bson.M{"_id": bookingId}, bson.M{"$set": bson.M{"status": status}})
	if err != nil {
		return fmt.Errorf("%w: could not update booking status for bookingId=%s: %v",
			dbErrors.ErrBookingRepo, bookingId.Hex(), err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("%w: could not find booking for bookingId=%s", dbErrors.ErrBookingRepo, bookingId.Hex())
	}

	return nil
}
