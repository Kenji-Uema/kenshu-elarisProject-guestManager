package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/message"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

type StayHandler struct {
	notificationService app.NotificationService
	cleaningService     app.CleaningService
	dayChangeEvent      port.MqConsumer
	writer              *message.Writer
	reader              *message.Reader
}

func NewStayHandler(notificationService app.NotificationService, dayChangeEvent port.MqConsumer,
	writer *message.Writer, reader *message.Reader) *StayHandler {
	return &StayHandler{
		notificationService: notificationService,
		dayChangeEvent:      dayChangeEvent,
		writer:              writer,
		reader:              reader,
	}
}

func (h StayHandler) Handle(ctx context.Context, booking domain.Booking) error {
	notificationCtx, stopNotifications := context.WithCancel(ctx)
	defer stopNotifications()

	dayChangeEvents, err := h.dayChangeEvent.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "stay handler: consume day change events", "error", err)
		return err
	}

	go func() {
		err := h.NotifyBreakfast(notificationCtx)
		if err != nil {
			slog.ErrorContext(notificationCtx, "stay handler: notify breakfast", "error", err)
		}
	}()
	go func() {
		err := h.NotifyDinner(notificationCtx)
		if err != nil {
			slog.ErrorContext(notificationCtx, "stay handler: notify dinner", "error", err)
		}
	}()

	for {
		if err := h.PrepareCottageForSleep(ctx, booking.CottageName); err != nil {
			slog.ErrorContext(ctx, "stay handler: prepare cottage for sleep", "error", err)
			return err
		}

		if err := h.CleanCottage(ctx, booking.CottageName); err != nil {
			slog.ErrorContext(ctx, "stay handler: clean cottage", "error", err)
			return err
		}

		timeEvent, err := h.waitDayChangeEvent(ctx, dayChangeEvents)
		if err != nil {
			slog.ErrorContext(ctx, "stay handler: wait day change event", "error", err)
			return err
		}

		if isCheckoutToday(booking, timeEvent) {
			break
		}
	}

	if err := h.NotifyCheckoutToday(ctx); err != nil {
		slog.ErrorContext(ctx, "stay handler: notify checkout today", "error", err)
		return err
	}

	stopNotifications()

	return nil
}

func isCheckoutToday(booking domain.Booking, timeEvent time.Time) bool {
	today := timeEvent.UTC()
	checkout := booking.StayPeriod.End.UTC()

	return checkout.Year() == today.Year() &&
		checkout.Month() == today.Month() &&
		checkout.Day() == today.Day()
}

func (h StayHandler) waitDayChangeEvent(ctx context.Context, events <-chan amqp.Delivery) (time.Time, error) {
	select {
	case <-ctx.Done():
		return time.Time{}, ctx.Err()
	case delivery, ok := <-events:
		if !ok {
			return time.Time{}, nil
		}

		timeEvent, err := parseDayChangeEvent(delivery.Body)
		if err != nil {
			nackDelivery(ctx, delivery, false)
			return time.Time{}, err
		}

		ackDelivery(ctx, delivery)
		return timeEvent, nil
	}
}

func parseDayChangeEvent(body []byte) (time.Time, error) {
	var timeEvent dto.TimeEvent
	if err := proto.Unmarshal(body, &timeEvent); err != nil {
		return time.Time{}, err
	}
	if timeEvent.GetTime() == nil {
		return time.Time{}, errors.New("missing time")
	}

	return timeEvent.GetTime().AsTime(), nil
}

func ackDelivery(ctx context.Context, delivery amqp.Delivery) {
	if err := delivery.Ack(false); err != nil {
		slog.ErrorContext(ctx, "stay handler: ack delivery", "error", err)
	}
}

func nackDelivery(ctx context.Context, delivery amqp.Delivery, requeue bool) {
	if err := delivery.Nack(false, requeue); err != nil {
		slog.ErrorContext(ctx, "stay handler: nack delivery", "error", err, "requeue", requeue)
	}
}

func (h StayHandler) CleanCottage(ctx context.Context, cottageName string) error {
	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION)
	if err != nil {
		slog.WarnContext(ctx, "wait for LEAVE_CLEANUP_NOTIFICATION", "error", err)
		return err
	}

	slog.InfoContext(ctx, "received LEAVE_CLEANUP_NOTIFICATION", "message", msg)

	cleaningRequest, err := domain.NewCleaningRequest(cottageName, "clean")
	if err != nil {
		return err
	}

	if err := h.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		return err
	}

	return nil
}

func (h StayHandler) PrepareCottageForSleep(ctx context.Context, cottageName string) error {
	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_GO_FOR_DINNER)
	if err != nil {
		slog.WarnContext(ctx, "wait for LEAVE_CLEANUP_NOTIFICATION", "error", err)
		return err
	}

	slog.InfoContext(ctx, "received GO_FOR_DINNER", "message", msg)

	cleaningRequest, err := domain.NewCleaningRequest(cottageName, "sleep")
	if err != nil {
		return err
	}

	if err := h.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		return err
	}

	return nil
}

func (h StayHandler) NotifyDinner(ctx context.Context) error {
	dinnerCh := h.notificationService.DinnerNotification(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-dinnerCh:
			if !ok {
				return nil
			}
			if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_DINNER_READY); err != nil {
				return fmt.Errorf("stay handler: send dinner notification: %w", err)
			}
		}
	}
}

func (h StayHandler) NotifyBreakfast(ctx context.Context) error {
	breakfastCh := h.notificationService.BreakfastNotification(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-breakfastCh:
			if !ok {
				return nil
			}

			if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_BREAKFAST_READY); err != nil {
				return fmt.Errorf("stay handler: send breakfast notification: %w", err)
			}
		}
	}
}

func (h StayHandler) NotifyCheckoutToday(ctx context.Context) error {
	checkOutCh := h.notificationService.CheckOutNotification()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case checkOutBookings, ok := <-checkOutCh:
		if !ok || len(checkOutBookings) == 0 {
			return nil
		}
	}

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_CHECK_OUT_TODAY); err != nil {
		return fmt.Errorf("stay handler: send checkout notification: %w", err)
	}

	return nil
}
