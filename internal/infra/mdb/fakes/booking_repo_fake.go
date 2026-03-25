package fakes

import (
	"context"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain/documents"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ port.BookingRepo = (*FakeBookingRepo)(nil)

type FakeBookingRepo struct {
	FindByGuestIdFn                 func(ctx context.Context, guestId bson.ObjectID) ([]documents.Booking, error)
	FindByCheckInDateFn             func(ctx context.Context, date time.Time) ([]documents.Booking, error)
	FindByGuestIdAndCheckInFn       func(ctx context.Context, guestId bson.ObjectID, checkIn time.Time) (documents.Booking, error)
	FindByGuestIdAndBookingNumberFn func(ctx context.Context, guestId bson.ObjectID, bookingNumber string) (documents.Booking, error)
	UpdateStatusFn                  func(ctx context.Context, bookingId bson.ObjectID, status enum.BookingStatus) error

	FindByGuestIdCallCount                 int
	FindByCheckInDateCallCount             int
	FindByGuestIdAndCheckInCallCount       int
	FindByGuestIdAndBookingNumberCallCount int
	UpdateStatusCallCount                  int

	LastFindByGuestIdCtx                        context.Context
	LastFindByGuestId                           bson.ObjectID
	LastFindByCheckInDateCtx                    context.Context
	LastFindByCheckInDate                       time.Time
	LastFindByGuestIdAndCheckInCtx              context.Context
	LastFindByGuestIdAndCheckInGuestId          bson.ObjectID
	LastFindByGuestIdAndCheckInDate             time.Time
	LastFindByGuestIdAndBookingNumberCtx        context.Context
	LastFindByGuestIdAndBookingNumberGuestId    bson.ObjectID
	LastFindByGuestIdAndBookingNumberBookingNum string
	LastUpdateStatusCtx                         context.Context
	LastUpdateStatusBookingId                   bson.ObjectID
	LastUpdateStatus                            enum.BookingStatus
}

func (f *FakeBookingRepo) FindByGuestId(ctx context.Context, guestId bson.ObjectID) ([]documents.Booking, error) {
	f.FindByGuestIdCallCount++
	f.LastFindByGuestIdCtx = ctx
	f.LastFindByGuestId = guestId

	if f.FindByGuestIdFn != nil {
		return f.FindByGuestIdFn(ctx, guestId)
	}

	return nil, nil
}

func (f *FakeBookingRepo) FindByCheckInDate(ctx context.Context, date time.Time) ([]documents.Booking, error) {
	f.FindByCheckInDateCallCount++
	f.LastFindByCheckInDateCtx = ctx
	f.LastFindByCheckInDate = date

	if f.FindByCheckInDateFn != nil {
		return f.FindByCheckInDateFn(ctx, date)
	}

	return nil, nil
}

func (f *FakeBookingRepo) FindByGuestIdAndCheckIn(ctx context.Context, guestId bson.ObjectID, checkIn time.Time) (documents.Booking, error) {
	f.FindByGuestIdAndCheckInCallCount++
	f.LastFindByGuestIdAndCheckInCtx = ctx
	f.LastFindByGuestIdAndCheckInGuestId = guestId
	f.LastFindByGuestIdAndCheckInDate = checkIn

	if f.FindByGuestIdAndCheckInFn != nil {
		return f.FindByGuestIdAndCheckInFn(ctx, guestId, checkIn)
	}

	return documents.Booking{}, nil
}

func (f *FakeBookingRepo) FindByGuestIdAndBookingNumber(ctx context.Context, guestId bson.ObjectID, bookingNumber string) (documents.Booking, error) {
	f.FindByGuestIdAndBookingNumberCallCount++
	f.LastFindByGuestIdAndBookingNumberCtx = ctx
	f.LastFindByGuestIdAndBookingNumberGuestId = guestId
	f.LastFindByGuestIdAndBookingNumberBookingNum = bookingNumber

	if f.FindByGuestIdAndBookingNumberFn != nil {
		return f.FindByGuestIdAndBookingNumberFn(ctx, guestId, bookingNumber)
	}

	return documents.Booking{}, nil
}

func (f *FakeBookingRepo) UpdateStatus(ctx context.Context, bookingId bson.ObjectID, status enum.BookingStatus) error {
	f.UpdateStatusCallCount++
	f.LastUpdateStatusCtx = ctx
	f.LastUpdateStatusBookingId = bookingId
	f.LastUpdateStatus = status

	if f.UpdateStatusFn != nil {
		return f.UpdateStatusFn(ctx, bookingId, status)
	}

	return nil
}
