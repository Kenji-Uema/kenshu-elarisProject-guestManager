package fakes

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/domain"
)

type StayHandler struct {
	HandleFn func(ctx context.Context, booking domain.Booking) error

	HandleCallCount int

	LastHandleCtx     context.Context
	LastHandleBooking domain.Booking
}

func (f *StayHandler) Handle(ctx context.Context, booking domain.Booking) error {
	f.HandleCallCount++
	f.LastHandleCtx = ctx
	f.LastHandleBooking = booking

	if f.HandleFn != nil {
		return f.HandleFn(ctx, booking)
	}

	return nil
}
