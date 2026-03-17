package handler

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/transport/grpc/clock"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/message"
)

type CheckinHandler struct {
	receptionService app.ReceptionService
	clockEmu         clock.Emu
	writer           *message.Writer
	reader           *message.Reader
}

func NewCheckinHandler(receptionService app.ReceptionService, writer *message.Writer, reader *message.Reader) *CheckinHandler {
	return &CheckinHandler{receptionService: receptionService, writer: writer, reader: reader}
}

var ErrUnexpectedResponse = errors.New("unexpected websocket response payload")

func (h CheckinHandler) WaitForGuest(ctx context.Context) error {
	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_SHOW_FOR_CHECKIN)
	if err != nil {
		slog.WarnContext(ctx, "wait for SHOW_FOR_CHECKIN", "error", err)
		return err
	}

	slog.InfoContext(ctx, "received SHOW_FOR_CHECKIN", "message", msg)

	return nil
}

func (h CheckinHandler) Handle(ctx context.Context) (domain.Booking, error) {
	now, err := h.clockEmu.Now(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "checkin handler: clock emu now", "error", err)
		return domain.Booking{}, err
	}

	var documentID string
	if documentID, err = h.requestDocumentID(ctx); err != nil {
		slog.ErrorContext(ctx, "checkin handler: request document", "error", err)
		return domain.Booking{}, err
	}

	slog.InfoContext(ctx, "checkin handler: received document id", "document_id", documentID)

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_BOOKING_CHECKING); err != nil {
		slog.ErrorContext(ctx, "checkin handler: send notification", "error", err)
		return domain.Booking{}, err
	}

	booking, err := h.receptionService.CheckIn(ctx, documentID, *now)
	if err != nil {
		slog.ErrorContext(ctx, "checkin handler: checkin", "error", err)

		var bookingID string
		if bookingID, err = h.requestBookingID(ctx); err != nil {
			slog.ErrorContext(ctx, "checkin handler: request booking id", "error", err)
			return domain.Booking{}, err
		}
		slog.InfoContext(ctx, "checkin handler: received booking id", "booking_id", bookingID)

		if booking, err = h.receptionService.CheckinFallback(ctx, documentID, bookingID, *now); err != nil {
			slog.ErrorContext(ctx, "checkin handler: checkin fallback", "error", err)
			return domain.Booking{}, err
		}
		slog.InfoContext(ctx, "checkin handler: fallback checkin succeeded")
	}

	slog.InfoContext(ctx, "checkin handler: checkin succeeded")

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_CHECK_IN_COMPLETE); err != nil {
		slog.ErrorContext(ctx, "checkin handler: send notification", "error", err)
		return domain.Booking{}, err
	}

	if _, err := h.giveCottageKey(ctx); err != nil {
		slog.ErrorContext(ctx, "checkin handler: give cottage key", "error", err)
		return domain.Booking{}, err
	}

	return booking, nil
}

func (h CheckinHandler) requestDocumentID(ctx context.Context) (string, error) {
	response, err := h.writer.SendSystemRequest(ctx, dto.SystemRequest_REQUEST_DOCUMENT, dto.GuestAction_SHOW_DOCUMENT)
	if err != nil {
		return "", err
	}

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetShowDocument() == nil {
		return "", ErrUnexpectedResponse
	}
	return guestResponse.GetShowDocument().GetDocumentId(), nil
}

func (h CheckinHandler) requestBookingID(ctx context.Context) (string, error) {
	response, err := h.writer.SendSystemRequest(ctx, dto.SystemRequest_REQUEST_BOOKING_NUMBER, dto.GuestAction_SHOW_BOOKING_NUMBER)
	if err != nil {
		return "", err
	}

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetShowBookingNumber() == nil {
		return "", ErrUnexpectedResponse
	}
	return guestResponse.GetShowBookingNumber().GetBookingId(), nil
}

func (h CheckinHandler) giveCottageKey(ctx context.Context) (string, error) {
	response, err := h.writer.SendSystemRequest(ctx, dto.SystemRequest_GIVE_COTTAGE_KEY, dto.GuestAction_TAKE_COTTAGE_KEY)
	if err != nil {
		return "", err
	}

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetReceiveCottageKey() == nil {
		return "", ErrUnexpectedResponse
	}
	return guestResponse.GetReceiveCottageKey().GetCottageKeyId(), nil
}
