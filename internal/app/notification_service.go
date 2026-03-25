package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/port"
)

type NotificationService interface {
	HourNotification(ctx context.Context, timerCh chan interface{}, hour int)
	CheckOutNotification(ctx context.Context, bookingCh chan []domain.Booking)
}
type notificationService struct {
	timeEventService TimeEventService
	cache            port.Cache
}

func NewNotificationService(timeEventService TimeEventService, cache port.Cache) NotificationService {
	return &notificationService{timeEventService: timeEventService, cache: cache}
}

func (n notificationService) HourNotification(ctx context.Context, timerCh chan interface{}, hour int) {
	events := make(chan time.Time, 1)
	n.timeEventService.Register(TimeEventHourChange, events)
	defer n.timeEventService.Unregister(TimeEventHourChange, events)

	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "notification service: hour notification context canceled")
			return
		case eventTime, ok := <-events:
			if !ok {
				slog.DebugContext(ctx, "notification service: hour change channel closed")
				return
			}
			if eventTime.Hour() == hour {
				select {
				case timerCh <- eventTime:
					slog.DebugContext(ctx, "notification service: published hour notification",
						"event_time", eventTime)
				case <-ctx.Done():
					slog.DebugContext(ctx, "notification service: canceled while publishing hour notification",
						"error", ctx.Err())
					return
				}
			}
		}
	}
}

func (n notificationService) CheckOutNotification(ctx context.Context, bookingCh chan []domain.Booking) {
	events := make(chan time.Time, 1)
	n.timeEventService.Register(TimeEventDayChange, events)
	defer n.timeEventService.Unregister(TimeEventDayChange, events)
	slog.DebugContext(ctx, "notification service: subscribed to day change notifications for checkout")

	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "notification service: checkout notification context canceled",
				"error", ctx.Err())
			return
		case eventTime, ok := <-events:
			if !ok {
				slog.DebugContext(ctx, "notification service: day change channel closed for checkout")
				return
			}

			redisKey := fmt.Sprintf("checkout.%s", eventTime.AddDate(0, 0, 1).UTC().Format("2006-01-02"))

			payload, err := n.cache.GetBytes(ctx, redisKey)
			if err != nil {
				if errors.Is(err, port.ErrCacheMiss) {
					slog.DebugContext(ctx, "notification service: no checkout bookings in redis for next day",
						"redis_key", redisKey, "event_time", eventTime)
					continue
				}
				slog.WarnContext(ctx, "notification service: failed to read checkout bookings from redis",
					"redis_key", redisKey, "event_time", eventTime, "error", err)
				continue
			}

			var checkOutBookings []domain.Booking
			if err := json.Unmarshal(payload, &checkOutBookings); err != nil {
				slog.WarnContext(ctx, "notification service: failed to decode checkout bookings from redis",
					"redis_key", redisKey, "event_time", eventTime, "error", err)
				continue
			}

			select {
			case bookingCh <- checkOutBookings:
				slog.DebugContext(ctx, "notification service: published checkout notification",
					"redis_key", redisKey, "event_time", eventTime, "booking_count", len(checkOutBookings))
			case <-ctx.Done():
				slog.DebugContext(ctx, "notification service: canceled while publishing checkout notification",
					"redis_key", redisKey, "error", ctx.Err())
				return
			}
		}
	}
}
