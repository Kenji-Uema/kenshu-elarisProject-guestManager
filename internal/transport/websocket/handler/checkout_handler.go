package handler

import (
	"context"

	"github.com/Kenji-Uema/guestManager/internal/app"
	"github.com/Kenji-Uema/guestManager/internal/domain"
	"github.com/Kenji-Uema/guestManager/internal/domain/dto"
	"github.com/Kenji-Uema/guestManager/internal/port"
	"github.com/Kenji-Uema/guestManager/internal/transport/websocket/chat"
)

type CheckoutHandler struct {
	receptionService app.ReceptionService
	clock            port.ClockClient
	writer           chat.Writer
	reader           chat.Reader
}

func NewCheckoutHandler(receptionService app.ReceptionService, clock port.ClockClient, writer chat.Writer, reader chat.Reader) *CheckoutHandler {
	return &CheckoutHandler{receptionService: receptionService, clock: clock, writer: writer, reader: reader}
}

func (h CheckoutHandler) Handle(ctx context.Context, booking domain.Booking) error {
	now, err := h.clock.Now(ctx)
	if err != nil {
		logHandlerError(ctx, "clock_read_failed", "error", err)
		return err
	}

	msg, err := h.reader.WaitForGuestAction(ctx, dto.GuestAction_PROCEED_TO_CHECKOUT)
	if err != nil {
		logHandlerWarn(ctx, "proceed_to_checkout_wait_failed", "error", err)
		return err
	}
	ctx = chat.ContextFromMessage(ctx, msg)
	logHandlerInfo(ctx, "checkout_started", "cottage_name", booking.CottageName)

	ctx, keyNumber, err := h.requestCottageKey(ctx)
	if err != nil {
		return err
	}
	logHandlerInfo(ctx, "cottage_key_requested", "cottage_name", booking.CottageName)
	if msg, err = h.reader.WaitForGuestAction(ctx, dto.GuestAction_RETURN_COTTAGE_KEY); err != nil {
		logHandlerError(ctx, "return_cottage_key_wait_failed", "cottage_name", booking.CottageName, "error", err)
		return err
	}
	ctx = chat.ContextFromMessage(ctx, msg)
	logHandlerInfo(ctx, "cottage_key_returned", "cottage_name", booking.CottageName, "key_number", keyNumber)
	if err := h.receptionService.ReturnCottageKey(ctx, booking.CottageName, keyNumber); err != nil {
		logHandlerError(ctx, "cottage_key_return_failed", "cottage_name", booking.CottageName, "key_number", keyNumber, "error", err)
		return err
	}

	if err := h.receptionService.CheckOut(ctx, booking, *now); err != nil {
		logHandlerError(ctx, "checkout_failed", "cottage_name", booking.CottageName, "error", err)
		return err
	}
	logHandlerInfo(ctx, "checkout_succeeded", "cottage_name", booking.CottageName)

	if err := h.writer.SendSystemNotification(ctx, dto.SystemNotification_CHECK_OUT_COMPLETE); err != nil {
		logHandlerError(ctx, "checkout_complete_notification_failed", "cottage_name", booking.CottageName, "error", err)
		return err
	}

	return nil
}

func (h CheckoutHandler) requestCottageKey(ctx context.Context) (context.Context, string, error) {
	response, err := h.writer.SendSystemRequest(ctx, dto.SystemRequest_REQUEST_COTTAGE_KEY, &dto.GuestResponse{
		Payload: &dto.GuestResponse_ReturnCottageKey{},
	})
	if err != nil {
		return ctx, "", err
	}
	responseCtx := chat.ContextFromMessage(ctx, response)

	guestResponse := response.GetGuestResponse()
	if guestResponse == nil || guestResponse.GetReturnCottageKey() == nil {
		return responseCtx, "", ErrUnexpectedResponse
	}
	return responseCtx, guestResponse.GetReturnCottageKey().GetCottageKeyId(), nil
}
