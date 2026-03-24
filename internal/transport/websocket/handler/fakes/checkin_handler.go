package fakes

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/domain"
)

type CheckinHandler struct {
	HandleFn func(ctx context.Context) (domain.Booking, error)

	HandleCallCount int

	LastHandleCtx context.Context
}

func (f *CheckinHandler) Handle(ctx context.Context) (domain.Booking, error) {
	f.HandleCallCount++
	f.LastHandleCtx = ctx

	if f.HandleFn != nil {
		return f.HandleFn(ctx)
	}

	return domain.Booking{}, nil
}
