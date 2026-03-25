package fakes

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
)

var _ app.CleaningService = (*CleaningService)(nil)

type CleaningService struct {
	CleanRoomFn func(ctx context.Context, request domain.CleaningOrder) error

	CleanRoomCallCount int

	LastCleanRoomCtx    context.Context
	LastCleaningRequest domain.CleaningOrder
}

func (f *CleaningService) CleanRoom(ctx context.Context, request domain.CleaningOrder) error {
	f.CleanRoomCallCount++
	f.LastCleanRoomCtx = ctx
	f.LastCleaningRequest = request

	if f.CleanRoomFn != nil {
		return f.CleanRoomFn(ctx, request)
	}

	return nil
}
