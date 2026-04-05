package handler

import (
	"context"
	"errors"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat"
)

type CheckinHandler struct {
	receptionService app.ReceptionService
	clock            port.ClockClient
	writer           chat.Writer
	reader           chat.Reader
}

func NewCheckinHandler(receptionService app.ReceptionService, clock port.ClockClient, writer chat.Writer, reader chat.Reader) *CheckinHandler {

	return &CheckinHandler{
		receptionService: receptionService,
		clock:            clock,
		writer:           writer,
		reader:           reader,
	}
}

var ErrUnexpectedResponse = errors.New("unexpected websocket response payload")

func (h CheckinHandler) Handle(ctx context.Context) (domain.Booking, error) {
	guestCtx, err := h.waitForGuest(ctx)
	if err != nil {
		return domain.Booking{}, err
	}
	ctx = guestCtx

	now, err := h.clock.Now(ctx)
	if err != nil {
		logHandlerError(ctx, "clock_read_failed", "error", err)
		return domain.Booking{}, err
	}

	var documentID string
	if ctx, documentID, err = h.requestDocumentID(ctx); err != nil {
		logHandlerError(ctx, "document_request_failed", "error", err)
		return domain.Booking{}, err
	}

	logHandlerInfo(ctx, "document_id_received", "document_id", documentID)

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_BOOKING_CHECKING); err != nil {
		logHandlerError(ctx, "booking_checking_notification_failed", "error", err)
		return domain.Booking{}, err
	}

	booking, err := h.receptionService.CheckIn(ctx, documentID, *now)
	if err != nil {
		logHandlerWarn(ctx, "checkin_fallback_started", "document_id", documentID, "error", err)

		var bookingID string
		if ctx, bookingID, err = h.requestBookingID(ctx); err != nil {
			logHandlerError(ctx, "booking_id_request_failed", "document_id", documentID, "error", err)
			return domain.Booking{}, err
		}
		logHandlerInfo(ctx, "booking_id_received", "booking_id", bookingID)

		if booking, err = h.receptionService.CheckinFallback(ctx, documentID, bookingID, *now); err != nil {
			logHandlerError(ctx, "checkin_fallback_failed", "document_id", documentID, "booking_id", bookingID, "error", err)
			return domain.Booking{}, err
		}
		logHandlerInfo(ctx, "checkin_fallback_succeeded", "document_id", documentID, "booking_id", bookingID, "cottage_name", booking.CottageName)
	}

	logHandlerInfo(ctx, "checkin_succeeded", "document_id", documentID, "cottage_name", booking.CottageName)

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_CHECK_IN_COMPLETE); err != nil {
		logHandlerError(ctx, "checkin_complete_notification_failed", "cottage_name", booking.CottageName, "error", err)
		return domain.Booking{}, err
	}

	ctx, keyNumber, err := h.giveCottageKey(ctx)
	if err != nil {
		logHandlerError(ctx, "cottage_key_request_failed", "cottage_name", booking.CottageName, "error", err)
		return domain.Booking{}, err
	}
	if _, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_TAKE_COTTAGE_KEY); err != nil {
		logHandlerError(ctx, "take_cottage_key_wait_failed", "cottage_name", booking.CottageName, "error", err)
		return domain.Booking{}, err
	}
	if err := h.receptionService.ReceiveCottageKey(ctx, booking.CottageName, keyNumber); err != nil {
		logHandlerError(ctx, "cottage_key_receive_failed", "cottage_name", booking.CottageName, "key_number", keyNumber, "error", err)
		return domain.Booking{}, err
	}
	logHandlerInfo(ctx, "cottage_key_received", "cottage_name", booking.CottageName, "key_number", keyNumber)

	return booking, nil
}

func (h CheckinHandler) waitForGuest(ctx context.Context) (context.Context, error) {
	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_SHOW_FOR_CHECKIN)
	if err != nil {
		logHandlerWarn(ctx, "show_for_checkin_wait_failed", "error", err)
		return ctx, err
	}

	msgCtx := chat.ContextFromMessage(ctx, msg)
	logHandlerInfo(msgCtx, "guest_action_received", guestActionAttrs(msg)...)

	return msgCtx, nil
}

func (h CheckinHandler) requestDocumentID(ctx context.Context) (context.Context, string, error) {
	response, err := h.writer.SendSystemRequest(ctx, dto.SystemRequest_REQUEST_DOCUMENT, &dto.GuestResponse{
		Payload: &dto.GuestResponse_ShowDocument{},
	})
	if err != nil {
		return ctx, "", err
	}
	responseCtx := chat.ContextFromMessage(ctx, response)

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetShowDocument() == nil {
		return responseCtx, "", ErrUnexpectedResponse
	}
	return responseCtx, guestResponse.GetShowDocument().GetDocumentId(), nil
}

func (h CheckinHandler) requestBookingID(ctx context.Context) (context.Context, string, error) {
	response, err := h.writer.SendSystemRequest(ctx, dto.SystemRequest_REQUEST_BOOKING_NUMBER, &dto.GuestResponse{
		Payload: &dto.GuestResponse_ShowBookingNumber{},
	})
	if err != nil {
		return ctx, "", err
	}
	responseCtx := chat.ContextFromMessage(ctx, response)

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetShowBookingNumber() == nil {
		return responseCtx, "", ErrUnexpectedResponse
	}
	return responseCtx, guestResponse.GetShowBookingNumber().GetBookingId(), nil
}

func (h CheckinHandler) giveCottageKey(ctx context.Context) (context.Context, string, error) {
	response, err := h.writer.SendSystemRequest(ctx, dto.SystemRequest_GIVE_COTTAGE_KEY, &dto.GuestResponse{
		Payload: &dto.GuestResponse_ReceiveCottageKey{},
	})
	if err != nil {
		return ctx, "", err
	}
	responseCtx := chat.ContextFromMessage(ctx, response)

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetReceiveCottageKey() == nil {
		return responseCtx, "", ErrUnexpectedResponse
	}
	return responseCtx, guestResponse.GetReceiveCottageKey().GetCottageKeyId(), nil
}
