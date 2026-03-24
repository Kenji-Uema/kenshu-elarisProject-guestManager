package fakes

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
)

type Reader struct {
	WaitForGuestActionFn func(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error)
	AckGuestActionFn     func(ctx context.Context) error

	WaitForGuestActionCallCount int
	AckGuestActionCallCount     int

	LastWaitForGuestActionCtx    context.Context
	LastWaitForGuestActionAction dto.GuestAction
	LastAckGuestActionCtx        context.Context
}

func (f *Reader) WaitForGuestAction(ctx context.Context, action dto.GuestAction) (*dto.ChatMessage, error) {
	f.WaitForGuestActionCallCount++
	f.LastWaitForGuestActionCtx = ctx
	f.LastWaitForGuestActionAction = action

	if f.WaitForGuestActionFn != nil {
		return f.WaitForGuestActionFn(ctx, action)
	}

	return nil, nil
}

func (f *Reader) AckGuestAction(ctx context.Context) error {
	f.AckGuestActionCallCount++
	f.LastAckGuestActionCtx = ctx

	if f.AckGuestActionFn != nil {
		return f.AckGuestActionFn(ctx)
	}

	return nil
}
