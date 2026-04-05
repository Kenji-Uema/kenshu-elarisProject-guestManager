package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app/validation"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var notificationServiceTracer = otel.Tracer("guest-manager.app.notification-service")

type NotificationService interface {
	HourNotification(ctx context.Context, timerCh chan interface{}, hour int)
	CheckOutNotification(ctx context.Context, bookingCh chan []domain.Booking)
}
type notificationService struct {
	timeEventService TimeEventService
	cache            port.Cache
}

func NewNotificationService(timeEventService TimeEventService, cache port.Cache) (NotificationService, error) {
	if timeEventService == nil {
		return nil, fmt.Errorf("NewNotificationService: timeEventService is required")
	}
	if cache == nil {
		return nil, fmt.Errorf("NewNotificationService: cache is required")
	}

	if err := validation.New().
		NotZeroValue("timeEventService", timeEventService).
		NotZeroValue("cache", cache).
		Validate(); err != nil {
		return nil, fmt.Errorf("NewNotificationService: %w", err)
	}

	return &notificationService{timeEventService: timeEventService, cache: cache}, nil
}

func (n notificationService) HourNotification(ctx context.Context, timerCh chan interface{}, hour int) {
	ctx, span := notificationServiceTracer.Start(ctx, "NotificationService.HourNotification")
	defer span.End()
	span.SetAttributes(attribute.Int("notification.hour", hour))

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
					span.AddEvent("hour_notification_published")
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
	ctx, span := notificationServiceTracer.Start(ctx, "NotificationService.CheckOutNotification")
	defer span.End()

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
					span.AddEvent("checkout_cache_miss")
					slog.DebugContext(ctx, "notification service: no checkout bookings in redis for next day",
						"redis_key", redisKey, "event_time", eventTime)
					continue
				}
				span.RecordError(err)
				slog.WarnContext(ctx, "notification service: failed to read checkout bookings from redis",
					"redis_key", redisKey, "event_time", eventTime, "error", err)
				continue
			}

			var checkOutBookings []domain.Booking
			if err := json.Unmarshal(payload, &checkOutBookings); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "checkout_payload_decode_failed")
				slog.WarnContext(ctx, "notification service: failed to decode checkout bookings from redis",
					"redis_key", redisKey, "event_time", eventTime, "error", err)
				continue
			}

			select {
			case bookingCh <- checkOutBookings:
				span.SetAttributes(
					attribute.String("cache.key", redisKey),
					attribute.Int("checkout.bookings.count", len(checkOutBookings)),
				)
				span.AddEvent("checkout_notification_published")
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
