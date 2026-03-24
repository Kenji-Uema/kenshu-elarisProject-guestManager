package fakes

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
)

var _ app.NotificationService = (*NotificationService)(nil)

type NotificationService struct {
	HourNotificationFn     func(ctx context.Context, timerCh chan interface{}, hour int)
	CheckOutNotificationFn func(ctx context.Context, bookingCh chan []domain.Booking)

	HourNotificationCallCount     int
	CheckOutNotificationCallCount int

	LastHourNotificationCtx     context.Context
	LastHourNotificationTimerCh chan interface{}
	LastHourNotificationHour    int
	LastCheckOutNotificationCtx context.Context
	LastCheckOutNotificationCh  chan []domain.Booking
}

func (f *NotificationService) HourNotification(ctx context.Context, timerCh chan interface{}, hour int) {
	f.HourNotificationCallCount++
	f.LastHourNotificationCtx = ctx
	f.LastHourNotificationTimerCh = timerCh
	f.LastHourNotificationHour = hour

	if f.HourNotificationFn != nil {
		f.HourNotificationFn(ctx, timerCh, hour)
	}
}

func (f *NotificationService) CheckOutNotification(ctx context.Context, bookingCh chan []domain.Booking) {
	f.CheckOutNotificationCallCount++
	f.LastCheckOutNotificationCtx = ctx
	f.LastCheckOutNotificationCh = bookingCh

	if f.CheckOutNotificationFn != nil {
		f.CheckOutNotificationFn(ctx, bookingCh)
	}
}
