package fakes

import (
	"context"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ app.GuestService = (*GuestService)(nil)

type GuestService struct {
	GetByIdFn          func(ctx context.Context, id bson.ObjectID) (domain.Guest, error)
	GetByDocumentFn    func(ctx context.Context, documentId string) (domain.Guest, error)
	AddFn              func(ctx context.Context, guest domain.Guest) (bson.ObjectID, error)
	UpdateFn           func(ctx context.Context, id bson.ObjectID, guest domain.Guest) (domain.Guest, error)
	GetBookingsFn      func(ctx context.Context, guestId bson.ObjectID) ([]domain.Booking, error)
	GetBookingByDateFn func(ctx context.Context, document string, date time.Time) (domain.Booking, error)

	GetByIdCallCount          int
	GetByDocumentCallCount    int
	AddCallCount              int
	UpdateCallCount           int
	GetBookingsCallCount      int
	GetBookingByDateCallCount int

	LastGetByIdCtx          context.Context
	LastGetById             bson.ObjectID
	LastGetByDocumentCtx    context.Context
	LastGetByDocument       string
	LastAddCtx              context.Context
	LastAddedGuest          domain.Guest
	LastUpdateCtx           context.Context
	LastUpdateId            bson.ObjectID
	LastUpdatedGuest        domain.Guest
	LastGetBookingsCtx      context.Context
	LastGetBookingsGuestId  bson.ObjectID
	LastGetBookingByDateCtx context.Context
	LastGetBookingDocument  string
	LastGetBookingDate      time.Time
}

func (f *GuestService) GetById(ctx context.Context, id bson.ObjectID) (domain.Guest, error) {
	f.GetByIdCallCount++
	f.LastGetByIdCtx = ctx
	f.LastGetById = id
	if f.GetByIdFn != nil {
		return f.GetByIdFn(ctx, id)
	}
	return domain.Guest{}, nil
}

func (f *GuestService) GetByDocument(ctx context.Context, documentId string) (domain.Guest, error) {
	f.GetByDocumentCallCount++
	f.LastGetByDocumentCtx = ctx
	f.LastGetByDocument = documentId
	if f.GetByDocumentFn != nil {
		return f.GetByDocumentFn(ctx, documentId)
	}
	return domain.Guest{}, nil
}

func (f *GuestService) Add(ctx context.Context, guest domain.Guest) (bson.ObjectID, error) {
	f.AddCallCount++
	f.LastAddCtx = ctx
	f.LastAddedGuest = guest
	if f.AddFn != nil {
		return f.AddFn(ctx, guest)
	}
	return bson.NilObjectID, nil
}

func (f *GuestService) Update(ctx context.Context, id bson.ObjectID, guest domain.Guest) (domain.Guest, error) {
	f.UpdateCallCount++
	f.LastUpdateCtx = ctx
	f.LastUpdateId = id
	f.LastUpdatedGuest = guest
	if f.UpdateFn != nil {
		return f.UpdateFn(ctx, id, guest)
	}
	return domain.Guest{}, nil
}

func (f *GuestService) GetBookings(ctx context.Context, guestId bson.ObjectID) ([]domain.Booking, error) {
	f.GetBookingsCallCount++
	f.LastGetBookingsCtx = ctx
	f.LastGetBookingsGuestId = guestId
	if f.GetBookingsFn != nil {
		return f.GetBookingsFn(ctx, guestId)
	}
	return nil, nil
}

func (f *GuestService) GetBookingByDate(ctx context.Context, document string, date time.Time) (domain.Booking, error) {
	f.GetBookingByDateCallCount++
	f.LastGetBookingByDateCtx = ctx
	f.LastGetBookingDocument = document
	f.LastGetBookingDate = date
	if f.GetBookingByDateFn != nil {
		return f.GetBookingByDateFn(ctx, document, date)
	}
	return domain.Booking{}, nil
}
