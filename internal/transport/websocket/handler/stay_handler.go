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
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat"
)

var errDayChangeStreamClosed = errors.New("day change stream closed")

type StayHandler struct {
	notificationService app.NotificationService
	cleaningService     app.CleaningService
	timeEventService    app.TimeEventService
	writer              chat.Writer
	reader              chat.Reader
}

func NewStayHandler(notificationService app.NotificationService, cleaningService app.CleaningService,
	timeEventService app.TimeEventService, writer chat.Writer, reader chat.Reader) *StayHandler {
	return &StayHandler{
		notificationService: notificationService,
		cleaningService:     cleaningService,
		timeEventService:    timeEventService,
		writer:              writer,
		reader:              reader,
	}
}

func (h StayHandler) Handle(ctx context.Context, booking domain.Booking) error {
	notificationCtx, stopNotifications := context.WithCancel(ctx)
	defer stopNotifications()

	dayChangeEvents := make(chan time.Time, 1)
	h.timeEventService.Register(app.TimeEventDayChange, dayChangeEvents)
	defer h.timeEventService.Unregister(app.TimeEventDayChange, dayChangeEvents)

	go func() {
		err := h.notifyBreakfast(notificationCtx)
		if err != nil {
			logHandlerError(notificationCtx, "breakfast_notification_failed", "error", err)
		}
	}()
	go func() {
		err := h.notifyDinner(notificationCtx)
		if err != nil {
			logHandlerError(notificationCtx, "dinner_notification_failed", "error", err)
		}
	}()

	if err := h.checkInDayRoutine(ctx, booking); err != nil {
		return err
	}

	if _, err := h.waitDayChangeEvent(ctx, dayChangeEvents); err != nil {
		logHandlerError(ctx, "day_change_wait_failed", "error", err)
		return err
	}

	for {
		if err := h.stayRoutine(ctx, booking); err != nil {
			return err
		}

		timeEvent, err := h.waitDayChangeEvent(ctx, dayChangeEvents)
		if err != nil {
			logHandlerError(ctx, "day_change_wait_failed", "error", err)
			return err
		}

		if isCheckoutToday(booking, timeEvent) {
			break
		}
	}

	if err := h.notifyCheckoutToday(ctx); err != nil {
		logHandlerError(ctx, "checkout_today_notification_failed", "error", err)
		return err
	}

	stopNotifications()

	if err := h.ignoreAction(ctx, dto.GuestAction_LEAVE_COTTAGE); err != nil {
		return err
	}

	return nil
}

func (h StayHandler) stayRoutine(ctx context.Context, booking domain.Booking) error {
	if err := h.ignoreAction(ctx, dto.GuestAction_WAKEUP); err != nil {
		return err
	}

	if err := h.ignoreAction(ctx, dto.GuestAction_GO_FOR_BREAKFAST); err != nil {
		return err
	}

	if err := h.cleanCottage(ctx, booking.CottageName); err != nil {
		logHandlerError(ctx, "cleaning_request_failed", "cottage_name", booking.CottageName, "error", err)
		return err
	}

	if err := h.ignoreAction(ctx, dto.GuestAction_ENJOY_RESORT); err != nil {
		return err
	}

	if err := h.ignoreAction(ctx, dto.GuestAction_GO_FOR_A_BATH); err != nil {
		return err
	}

	if err := h.prepareCottageForSleep(ctx, booking.CottageName); err != nil {
		logHandlerError(ctx, "prepare_for_sleep_failed", "cottage_name", booking.CottageName, "error", err)
		return err
	}

	if err := h.ignoreAction(ctx, dto.GuestAction_GO_TO_SLEEP); err != nil {
		return err
	}

	return nil
}

func (h StayHandler) checkInDayRoutine(ctx context.Context, booking domain.Booking) error {
	if err := h.ignoreAction(ctx, dto.GuestAction_ENTER_COTTAGE); err != nil {
		return err
	}

	if err := h.ignoreAction(ctx, dto.GuestAction_GO_FOR_A_BATH); err != nil {
		return err
	}

	if err := h.prepareCottageForSleep(ctx, booking.CottageName); err != nil {
		logHandlerError(ctx, "prepare_for_sleep_failed", "cottage_name", booking.CottageName, "error", err)
		return err
	}

	if err := h.ignoreAction(ctx, dto.GuestAction_GO_TO_SLEEP); err != nil {
		return err
	}

	return nil
}

func isCheckoutToday(booking domain.Booking, timeEvent time.Time) bool {
	today := timeEvent.UTC()
	checkout := booking.StayPeriod.CheckOut.UTC()

	return checkout.Year() == today.Year() &&
		checkout.Month() == today.Month() &&
		checkout.Day() == today.Day()
}

func (h StayHandler) waitDayChangeEvent(ctx context.Context, events <-chan time.Time) (time.Time, error) {
	select {
	case <-ctx.Done():
		return time.Time{}, ctx.Err()
	case timeEvent, ok := <-events:
		if !ok {
			return time.Time{}, errDayChangeStreamClosed
		}
		return timeEvent, nil
	}
}

func (h StayHandler) ignoreAction(ctx context.Context, action dto.GuestAction) error {
	logHandlerInfo(ctx, "guest_action_acknowledged", "action", action.String())

	return h.reader.AckGuestAction(ctx)
}

func (h StayHandler) cleanCottage(ctx context.Context, cottageName string) error {
	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION)
	if err != nil {
		logHandlerWarn(ctx, "leave_cleanup_notification_wait_failed", "error", err)
		return err
	}

	logHandlerInfo(ctx, "guest_action_received", guestActionAttrs(msg)...)

	cleaningRequest, err := domain.NewCleaningOrder(cottageName, enum.FullCleaning)
	if err != nil {
		return err
	}

	if err := h.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		return err
	}
	logHandlerInfo(ctx, "cleaning_requested", "cottage_name", cottageName, "request_type", string(enum.FullCleaning))

	return nil
}

func (h StayHandler) prepareCottageForSleep(ctx context.Context, cottageName string) error {
	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_GO_FOR_DINNER)
	if err != nil {
		logHandlerWarn(ctx, "go_for_dinner_wait_failed", "error", err)
		return err
	}

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
	dinnerCh := make(chan interface{})
	go h.notificationService.HourNotification(ctx, dinnerCh, 18)

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

func (h StayHandler) notifyBreakfast(ctx context.Context) error {
	breakfastCh := make(chan interface{})
	go h.notificationService.HourNotification(ctx, breakfastCh, 6)

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

func (h StayHandler) notifyCheckoutToday(ctx context.Context) error {
	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_CHECK_OUT_TODAY); err != nil {
		return fmt.Errorf("stay handler: send checkout notification: %w", err)
	}

	return nil
}
