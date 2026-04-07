package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/domain/enum"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var errDayChangeStreamClosed = errors.New("day change stream closed")
var stayHandlerTracer = otel.Tracer("guest-manager.websocket.stay-handler")

type StayHandler struct {
	notificationService app.NotificationService
	cleaningService     app.CleaningService
	timeEventService    app.TimeEventService
	clock               port.ClockClient
	writer              chat.Writer
	reader              chat.Reader
}

func NewStayHandler(notificationService app.NotificationService, cleaningService app.CleaningService,
	timeEventService app.TimeEventService, clock port.ClockClient, writer chat.Writer, reader chat.Reader) *StayHandler {
	return &StayHandler{
		notificationService: notificationService,
		cleaningService:     cleaningService,
		timeEventService:    timeEventService,
		clock:               clock,
		writer:              writer,
		reader:              reader,
	}
}

func (h StayHandler) Handle(ctx context.Context, booking domain.Booking) error {
	ctx, span := stayHandlerTracer.Start(ctx, "StayHandler.Handle")
	defer span.End()
	span.SetAttributes(
		attribute.String("booking.id", booking.Id.Hex()),
		attribute.String("booking.cottage_name", booking.CottageName),
	)

	dayChangeEvents := make(chan time.Time, 1)
	h.timeEventService.Register(app.TimeEventDayChange, dayChangeEvents)
	defer h.timeEventService.Unregister(app.TimeEventDayChange, dayChangeEvents)

	if err := h.checkInDayRoutine(ctx, booking); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "checkin_day_routine_failed")
		return err
	}

	checkoutReached, err := h.waitForStayTransition(ctx, booking, dayChangeEvents)
	if err != nil {
		logHandlerError(ctx, "day_change_wait_failed", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "initial_day_change_wait_failed")
		return err
	}

	for !checkoutReached {
		if err := h.stayRoutine(ctx, booking); err != nil {
			return err
		}

		checkoutReached, err = h.waitForStayTransition(ctx, booking, dayChangeEvents)
		if err != nil {
			logHandlerError(ctx, "day_change_wait_failed", "error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "stay_day_change_wait_failed")
			return err
		}
	}

	if err := h.checkoutDayRoutine(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "checkout_day_routine_failed")
		return err
	}

	return nil
}

func (h StayHandler) stayRoutine(ctx context.Context, booking domain.Booking) error {
	ctx, span := stayHandlerTracer.Start(ctx, "StayHandler.stayRoutine")
	defer span.End()
	span.SetAttributes(
		attribute.String("booking.id", booking.Id.Hex()),
		attribute.String("booking.cottage_name", booking.CottageName),
	)

	if err := h.waitForAction(ctx, dto.GuestAction_WAKEUP); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "wakeup_wait_failed")
		return err
	}

	if err := h.notifyBreakfast(ctx); err != nil {
		logHandlerError(ctx, "breakfast_notification_failed", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "breakfast_notification_failed")
		return err
	}

	if err := h.waitForAction(ctx, dto.GuestAction_GO_FOR_BREAKFAST); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "breakfast_wait_failed")
		return err
	}

	if err := h.cleanCottage(ctx, booking.CottageName); err != nil {
		logHandlerError(ctx, "cleaning_request_failed", "cottage_name", booking.CottageName, "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "cleaning_request_failed")
		return err
	}

	if err := h.waitForAction(ctx, dto.GuestAction_ENJOY_RESORT); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "enjoy_resort_wait_failed")
		return err
	}

	if err := h.waitForAction(ctx, dto.GuestAction_GO_FOR_A_BATH); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "bath_wait_failed")
		return err
	}

	if err := h.notifyDinner(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "notify dinner failed")
		return err
	}

	if err := h.prepareCottageForSleep(ctx, booking.CottageName); err != nil {
		logHandlerError(ctx, "prepare_for_sleep_failed", "cottage_name", booking.CottageName, "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "prepare_for_sleep_failed")
		return err
	}

	if err := h.waitForAction(ctx, dto.GuestAction_GO_TO_SLEEP); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "sleep_wait_failed")
		return err
	}

	return nil
}

func (h StayHandler) checkInDayRoutine(ctx context.Context, booking domain.Booking) error {
	ctx, span := stayHandlerTracer.Start(ctx, "StayHandler.checkInDayRoutine")
	defer span.End()
	span.SetAttributes(
		attribute.String("booking.id", booking.Id.Hex()),
		attribute.String("booking.cottage_name", booking.CottageName),
	)

	if err := h.waitForAction(ctx, dto.GuestAction_ENTER_COTTAGE); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "enter_cottage_wait_failed")
		return err
	}

	if err := h.waitForAction(ctx, dto.GuestAction_GO_FOR_A_BATH); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "bath_wait_failed")
		return err
	}

	if err := h.notifyDinner(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "notify dinner failed")
		return err
	}

	if err := h.prepareCottageForSleep(ctx, booking.CottageName); err != nil {
		logHandlerError(ctx, "prepare_for_sleep_failed", "cottage_name", booking.CottageName, "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "prepare_for_sleep_failed")
		return err
	}

	if err := h.waitForAction(ctx, dto.GuestAction_GO_TO_SLEEP); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "sleep_wait_failed")
		return err
	}

	return nil
}

func (h StayHandler) checkoutDayRoutine(ctx context.Context) error {
	if err := h.waitForAction(ctx, dto.GuestAction_WAKEUP); err != nil {
		return err
	}

	if err := h.waitForAction(ctx, dto.GuestAction_LEAVE_COTTAGE); err != nil {
		return err
	}

	return nil
}

func isCheckoutToday(booking domain.Booking, timeEvent time.Time) bool {
	today := timeEvent.UTC()
	checkout := booking.StayPeriod.CheckOut.UTC()

	checkoutDay := time.Date(checkout.Year(), checkout.Month(), checkout.Day(), 0, 0, 0, 0, time.UTC)
	eventDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	return !eventDay.Before(checkoutDay)
}

func (h StayHandler) waitForStayTransition(ctx context.Context, booking domain.Booking, events <-chan time.Time) (bool, error) {
	if reached, err := h.isCheckoutReachedNow(ctx, booking); err == nil && reached {
		return true, nil
	}

	if h.clock == nil {
		timeEvent, err := h.waitDayChangeEvent(ctx, events)
		if err != nil {
			return false, err
		}

		return isCheckoutToday(booking, timeEvent), nil
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case timeEvent, ok := <-events:
			if !ok {
				return false, errDayChangeStreamClosed
			}
			return isCheckoutToday(booking, timeEvent), nil
		case <-ticker.C:
			reached, err := h.isCheckoutReachedNow(ctx, booking)
			if err != nil {
				logHandlerWarn(ctx, "clock_read_failed", "error", err)
				continue
			}
			if reached {
				return true, nil
			}
		}
	}
}

func (h StayHandler) isCheckoutReachedNow(ctx context.Context, booking domain.Booking) (bool, error) {
	if h.clock == nil {
		return false, nil
	}

	now, err := h.clock.Now(ctx)
	if err != nil {
		return false, err
	}
	if now == nil {
		return false, nil
	}

	return isCheckoutToday(booking, now.UTC()), nil
}

func (h StayHandler) waitDayChangeEvent(ctx context.Context, events <-chan time.Time) (time.Time, error) {
	ctx, span := stayHandlerTracer.Start(ctx, "StayHandler.waitDayChangeEvent")
	defer span.End()

	select {
	case <-ctx.Done():
		err := ctx.Err()
		span.RecordError(err)
		span.SetStatus(codes.Error, "context_canceled")
		return time.Time{}, err
	case timeEvent, ok := <-events:
		if !ok {
			span.RecordError(errDayChangeStreamClosed)
			span.SetStatus(codes.Error, "day_change_stream_closed")
			return time.Time{}, errDayChangeStreamClosed
		}
		span.SetAttributes(attribute.String("day_change.at", timeEvent.UTC().Format(time.RFC3339)))
		return timeEvent, nil
	}
}

func (h StayHandler) waitForAction(ctx context.Context, action dto.GuestAction) error {
	msg, err := h.reader.WaitForGuestAction(ctx, action)
	if err != nil {
		return err
	}
	msgCtx := chat.ContextFromMessage(ctx, msg)
	logHandlerInfo(msgCtx, "guest_action_received", guestActionAttrs(msg)...)
	return nil
}

func (h StayHandler) cleanCottage(ctx context.Context, cottageName string) error {
	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION)
	if err != nil {
		logHandlerWarn(ctx, "leave_cleanup_notification_wait_failed", "error", err)
		return err
	}
	ctx = chat.ContextFromMessage(ctx, msg)

	logHandlerInfo(ctx, "guest_action_received", guestActionAttrs(msg)...)

	cleaningRequest, err := domain.NewCleaningOrder(cottageName, enum.DailyCleaning)
	if err != nil {
		return err
	}

	if err := h.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		return err
	}
	logHandlerInfo(ctx, "cleaning_requested", "cottage_name", cottageName, "request_type", string(enum.DailyCleaning))

	return nil
}

func (h StayHandler) prepareCottageForSleep(ctx context.Context, cottageName string) error {
	if err := h.notifyDinner(ctx); err != nil {
		logHandlerError(ctx, "dinner_notification_failed", "error", err)
		return err
	}

	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_GO_FOR_DINNER)
	if err != nil {
		logHandlerWarn(ctx, "go_for_dinner_wait_failed", "error", err)
		return err
	}
	ctx = chat.ContextFromMessage(ctx, msg)

	logHandlerInfo(ctx, "guest_action_received", guestActionAttrs(msg)...)

	cleaningRequest, err := domain.NewCleaningOrder(cottageName, enum.PrepareForSleep)
	if err != nil {
		return err
	}

	if err := h.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		return err
	}
	logHandlerInfo(ctx, "cleaning_requested", "cottage_name", cottageName, "request_type", string(enum.PrepareForSleep))

	return nil
}

func (h StayHandler) notifyDinner(ctx context.Context) error {
	return h.notifyMeal(ctx, 18, dto.SystemNotification_DINNER_READY, "dinner")
}

func (h StayHandler) notifyBreakfast(ctx context.Context) error {
	return h.notifyMeal(ctx, 6, dto.SystemNotification_BREAKFAST_READY, "breakfast")
}

func (h StayHandler) notifyMeal(ctx context.Context, hour int, notification dto.SystemNotification, label string) error {
	timerCh := make(chan interface{}, 1)
	notificationCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go h.notificationService.HourNotification(notificationCtx, timerCh, hour)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case _, ok := <-timerCh:
		if !ok {
			return nil
		}
		if err := h.writer.SendSystemNotification(ctx, notification); err != nil {
			return fmt.Errorf("stay handler: send %s notification: %w", label, err)
		}
		return nil
	}
}

func (h StayHandler) notifyCheckoutToday(ctx context.Context) error {
	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_CHECK_OUT_TODAY); err != nil {
		return fmt.Errorf("stay handler: send checkout notification: %w", err)
	}

	return nil
}
