package fakes

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
)

type Writer struct {
	SendSystemRequestFn      func(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error)
	SendSystemNotificationFn func(ctx context.Context, notification dto.SystemNotification) error

	SendSystemRequestCallCount      int
	SendSystemNotificationCallCount int

	LastSendSystemRequestCtx          context.Context
	LastSendSystemRequestRequest      dto.SystemRequest
	LastSendSystemRequestResponseType *dto.GuestResponse
	LastSendSystemNotificationCtx     context.Context
	LastSendSystemNotificationValue   dto.SystemNotification
}

func (f *Writer) SendSystemRequest(ctx context.Context, request dto.SystemRequest, responseType *dto.GuestResponse) (*dto.ChatMessage, error) {
	f.SendSystemRequestCallCount++
	f.LastSendSystemRequestCtx = ctx
	f.LastSendSystemRequestRequest = request
	f.LastSendSystemRequestResponseType = responseType

	if f.SendSystemRequestFn != nil {
		return f.SendSystemRequestFn(ctx, request, responseType)
	}

	return nil, nil
}

func (f *Writer) SendSystemNotification(ctx context.Context, notification dto.SystemNotification) error {
	f.SendSystemNotificationCallCount++
	f.LastSendSystemNotificationCtx = ctx
	f.LastSendSystemNotificationValue = notification

	if f.SendSystemNotificationFn != nil {
		return f.SendSystemNotificationFn(ctx, notification)
	}

	return nil
}
