package fakes

import (
	"context"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
)

var _ app.ReceptionService = (*ReceptionService)(nil)

type ReceptionService struct {
	CheckInFn           func(ctx context.Context, guestDocument string, today time.Time) (domain.Booking, error)
	CheckOutFn          func(ctx context.Context, booking domain.Booking, today time.Time) error
	CheckinFallbackFn   func(ctx context.Context, document string, bookingNumber string, today time.Time) (domain.Booking, error)
	ReceiveCottageKeyFn func(ctx context.Context, cottageName string, keyNumber string) error
	ReturnCottageKeyFn  func(ctx context.Context, cottageName string, keyNumber string) error

	CheckInCallCount           int
	CheckOutCallCount          int
	CheckinFallbackCallCount   int
	ReceiveCottageKeyCallCount int
	ReturnCottageKeyCallCount  int

	LastCheckInCtx            context.Context
	LastCheckOutCtx           context.Context
	LastCheckinFallbackCtx    context.Context
	LastReceiveCottageKeyCtx  context.Context
	LastReturnCottageKeyCtx   context.Context
	LastCheckInGuestDocument  string
	LastCheckOutBooking       domain.Booking
	LastFallbackDocument      string
	LastFallbackBookingNumber string
	LastCheckInToday          time.Time
	LastCheckOutToday         time.Time
	LastFallbackToday         time.Time
	LastReceiveCottageName    string
	LastReceiveKeyNumber      string
	LastReturnCottageName     string
	LastReturnKeyNumber       string
}

func (f *ReceptionService) CheckIn(ctx context.Context, guestDocument string, today time.Time) (domain.Booking, error) {
	f.CheckInCallCount++
	f.LastCheckInCtx = ctx
	f.LastCheckInGuestDocument = guestDocument
	f.LastCheckInToday = today

	if f.CheckInFn != nil {
		return f.CheckInFn(ctx, guestDocument, today)
	}

	return domain.Booking{}, nil
}

func (f *ReceptionService) CheckOut(ctx context.Context, booking domain.Booking, today time.Time) error {
	f.CheckOutCallCount++
	f.LastCheckOutCtx = ctx
	f.LastCheckOutBooking = booking
	f.LastCheckOutToday = today

	if f.CheckOutFn != nil {
		return f.CheckOutFn(ctx, booking, today)
	}

	return nil
}

func (f *ReceptionService) CheckinFallback(ctx context.Context, document string, bookingNumber string, today time.Time) (domain.Booking, error) {
	f.CheckinFallbackCallCount++
	f.LastCheckinFallbackCtx = ctx
	f.LastFallbackDocument = document
	f.LastFallbackBookingNumber = bookingNumber
	f.LastFallbackToday = today

	if f.CheckinFallbackFn != nil {
		return f.CheckinFallbackFn(ctx, document, bookingNumber, today)
	}

	return domain.Booking{}, nil
}

func (f *ReceptionService) ReceiveCottageKey(ctx context.Context, cottageName string, keyNumber string) error {
	f.ReceiveCottageKeyCallCount++
	f.LastReceiveCottageKeyCtx = ctx
	f.LastReceiveCottageName = cottageName
	f.LastReceiveKeyNumber = keyNumber

	if f.ReceiveCottageKeyFn != nil {
		return f.ReceiveCottageKeyFn(ctx, cottageName, keyNumber)
	}

	return nil
}

func (f *ReceptionService) ReturnCottageKey(ctx context.Context, cottageName string, keyNumber string) error {
	f.ReturnCottageKeyCallCount++
	f.LastReturnCottageKeyCtx = ctx
	f.LastReturnCottageName = cottageName
	f.LastReturnKeyNumber = keyNumber

	if f.ReturnCottageKeyFn != nil {
		return f.ReturnCottageKeyFn(ctx, cottageName, keyNumber)
	}

	return nil
}
