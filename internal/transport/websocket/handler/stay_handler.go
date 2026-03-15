package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/message"
)

type StayHandler struct {
	notificationService app.NotificationService
	cleaningService     app.CleaningService
	writer              *message.Writer
	reader              *message.Reader
}

func NewStayHandler(notificationService app.NotificationService, writer *message.Writer, reader *message.Reader) *StayHandler {
	return &StayHandler{notificationService: notificationService, writer: writer, reader: reader}
}

func (h StayHandler) CleanCottage(ctx context.Context, roomNumber string) error {
	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_LEAVE_CLEANUP_NOTIFICATION)
	if err != nil {
		slog.WarnContext(ctx, "wait for LEAVE_CLEANUP_NOTIFICATION", "error", err)
		return err
	}

	slog.InfoContext(ctx, "received LEAVE_CLEANUP_NOTIFICATION", "message", msg)

	cleaningRequest, err := domain.NewCleaningRequest(roomNumber, "clean")
	if err != nil {
		return err
	}

	if err := h.cleaningService.CleanRoom(ctx, cleaningRequest); err != nil {
		return err
	}

	return nil
}

func (h StayHandler) PrepareCottageForSleep(ctx context.Context, roomNumber string) error {
	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_GO_FOR_DINNER)
	if err != nil {
		slog.WarnContext(ctx, "wait for LEAVE_CLEANUP_NOTIFICATION", "error", err)
		return err
	}

	slog.InfoContext(ctx, "received GO_FOR_DINNER", "message", msg)

	cleaningRequest, err := domain.NewCleaningRequest(roomNumber, "sleep")
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
	select {
	case <-ctx.Done():
		return ctx.Err()
	case _, ok := <-dinnerCh:
		if !ok {
			return nil
		}
	}

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_DINNER_READY); err != nil {
		return fmt.Errorf("stay handler: send dinner notification: %w", err)
	}

	return nil
}

func (h StayHandler) NotifyBreakfast(ctx context.Context) error {
	breakfastCh := h.notificationService.BreakfastNotification(ctx)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case _, ok := <-breakfastCh:
		if !ok {
			return nil
		}
	}

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_BREAKFAST_READY); err != nil {
		return fmt.Errorf("stay handler: send breakfast notification: %w", err)
	}

	return nil
}

func (h StayHandler) NotifyCheckoutTomorrow(ctx context.Context) error {
	checkOutCh := h.notificationService.CheckOutNotification()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case checkOutBookings, ok := <-checkOutCh:
		if !ok || len(checkOutBookings) == 0 {
			return nil
		}
	}

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_CHECK_OUT_TOMORROW); err != nil {
		return fmt.Errorf("stay handler: send checkout notification: %w", err)
	}

	return nil
}
